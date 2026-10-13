import argparse
import json
import os
from pathlib import Path
import select
import signal
import subprocess
import sys
import tempfile
import time

REPO = Path(__file__).resolve().parent.parent


def record(root, action, **details):
    with (root / 'actions.jsonl').open('a') as log:
        log.write(json.dumps({'time': time.time(), 'action': action, **details}) + '\n')


def control(root, *args, text=None):
    state = json.loads((root / 'state.json').read_text())
    result = subprocess.run(
        ['kitten', '@', '--to', state['socket'], *args],
        input=text, text=True, capture_output=True, timeout=15,
    )
    if result.returncode:
        raise RuntimeError(result.stderr or result.stdout)
    return result.stdout


def status(root):
    windows = []
    for os_window in json.loads(control(root, 'ls')):
        for tab in os_window['tabs']:
            for window in tab['windows']:
                windows.append({key: window.get(key) for key in (
                    'id', 'title', 'columns', 'lines', 'is_active', 'foreground_processes',
                )})
    return json.dumps(windows, indent=2)


def stop(root):
    state = json.loads((root / 'state.json').read_text())
    sessions = subprocess.run(
        ['zellij', 'list-sessions', '--short', '--no-formatting'],
        env=state['environment'], capture_output=True, text=True, timeout=10,
    )
    if sessions.returncode == 0:
        for name in sessions.stdout.splitlines():
            subprocess.run(['zellij', 'kill-session', name], env=state['environment'],
                           capture_output=True, timeout=10)
    for name in ('kitty_pid', 'xvfb_pid'):
        try:
            os.killpg(state[name], signal.SIGTERM)
        except ProcessLookupError:
            pass
    record(root, 'stop')


def start(editor):
    root = Path(tempfile.mkdtemp(prefix='cdvis-', dir=os.environ.get('CODEDRILL_QA_TMP')))
    print(root, flush=True)
    for name in ('home', 'config/codedrill', 'data', 'cache', 'runtime', 'zellij-config', 'screenshots'):
        (root / name).mkdir(parents=True, exist_ok=True, mode=0o700)
    (root / 'config/codedrill/config.json').write_text(json.dumps({
        'editor': editor, 'sound': False, 'countdown': 3,
    }))
    subprocess.run(['go', 'build', '-o', str(root / 'codedrill'), './cmd/codedrill'], cwd=REPO, check=True)
    read_fd, write_fd = os.pipe()
    with (root / 'xvfb.log').open('w') as log:
        xvfb = subprocess.Popen(
            ['Xvfb', '-displayfd', str(write_fd), '-screen', '0', '1920x1200x24', '-nolisten', 'tcp'],
            pass_fds=(write_fd,), stdout=log, stderr=log, start_new_session=True,
        )
    os.close(write_fd)
    try:
        report = b''
        deadline = time.monotonic() + 15
        while not report.endswith(b'\n'):
            ready, _, _ = select.select([read_fd], [], [], max(0, deadline - time.monotonic()))
            if not ready:
                raise RuntimeError('Xvfb did not report a display; inspect xvfb.log')
            chunk = os.read(read_fd, 100)
            if not chunk:
                raise RuntimeError('Xvfb closed its display pipe; inspect xvfb.log')
            report += chunk
        display = ':' + report.decode().strip()
    finally:
        os.close(read_fd)
    env = {k: v for k, v in os.environ.items()
           if not k.startswith(('ZELLIJ', 'KITTY', 'TMUX')) and k != 'WAYLAND_DISPLAY'}
    env.update({
        'DISPLAY': display, 'LIBGL_ALWAYS_SOFTWARE': '1',
        'HOME': str(root / 'home'), 'XDG_CONFIG_HOME': str(root / 'config'),
        'XDG_DATA_HOME': str(root / 'data'), 'XDG_CACHE_HOME': str(root / 'cache'),
        'XDG_RUNTIME_DIR': str(root / 'runtime'),
        'ZELLIJ_CONFIG_DIR': str(root / 'zellij-config'), 'ZELLIJ_SOCKET_DIR': str(root / 'z'),
        'CODEDRILL_EDITOR': editor, 'PS1': 'qa$ ', 'HISTFILE': '/dev/null',
    })
    socket = 'unix:' + str(root / 'kitty.sock')
    command = ['kitty', '--config', 'NONE', '--listen-on', socket, '--title', 'codedrill visual QA',
               '--directory', str(root)]
    for option in ('allow_remote_control=socket-only', 'linux_display_server=x11',
                   'remember_window_size=no', 'initial_window_width=120c', 'initial_window_height=40c',
                   'font_family=monospace', 'font_size=12', 'window_padding_width=0',
                   'enable_audio_bell=no', 'cursor_blink_interval=0', 'shell_integration=disabled'):
        command += ['--override', option]
    command += ['bash', '--noprofile', '--norc']
    with (root / 'kitty.log').open('w') as log:
        kitty = subprocess.Popen(command, env=env, stdout=log, stderr=log, start_new_session=True)
    state = {'socket': socket, 'display': display, 'kitty_pid': kitty.pid,
             'xvfb_pid': xvfb.pid, 'editor': editor, 'environment': env}
    (root / 'state.json').write_text(json.dumps(state, indent=2))
    record(root, 'start', editor=editor, command=command)
    for _ in range(100):
        if kitty.poll() is not None:
            stop(root)
            raise RuntimeError('Kitty exited; inspect kitty.log')
        try:
            print(status(root))
            return
        except (RuntimeError, subprocess.TimeoutExpired):
            time.sleep(.1)
    stop(root)
    raise RuntimeError('Kitty remote control did not become ready')


def snapshot(root, name):
    if not name or Path(name).name != name:
        raise ValueError('Use a single filename stem for screenshots')
    path = root / 'screenshots' / name
    text = control(root, 'get-text', '--extent', 'screen')
    path.with_suffix('.txt').write_text(text)
    path.with_suffix('.ansi').write_text(control(root, 'get-text', '--extent', 'screen', '--ansi'))
    path.with_suffix('.json').write_text(status(root))
    control(root, 'screenshot', str(path.with_suffix('.png')))
    record(root, 'snapshot', name=name)
    print(path.with_suffix('.png'))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('action', choices=('start', 'key', 'text', 'screen', 'wait', 'shot', 'resize', 'status', 'stop'))
    parser.add_argument('--session', type=Path)
    parser.add_argument('--editor', default='nvim')
    parser.add_argument('--timeout', type=float, default=20)
    parser.add_argument('values', nargs='*')
    args = parser.parse_intermixed_args()
    if args.action == 'start':
        start(args.editor)
        return
    if args.session is None:
        parser.error('--session is required after start')
    root = args.session.resolve()
    if args.action == 'key':
        for key in args.values:
            control(root, 'send-key', key)
            record(root, 'key', key=key)
    elif args.action == 'text':
        text = ' '.join(args.values) if args.values else sys.stdin.read()
        control(root, 'send-text', '--stdin', '--bracketed-paste', 'auto', text=text)
        record(root, 'text', text=text)
    elif args.action == 'screen':
        print(control(root, 'get-text', '--extent', 'screen'))
    elif args.action == 'status':
        print(status(root))
    elif args.action == 'shot':
        snapshot(root, args.values[0])
    elif args.action == 'resize':
        control(root, 'resize-os-window', '--unit', 'cells', '--width', args.values[0], '--height', args.values[1])
        record(root, 'resize', columns=int(args.values[0]), rows=int(args.values[1]))
        deadline = time.monotonic() + args.timeout
        while time.monotonic() < deadline:
            current = status(root)
            if all(w['columns'] == int(args.values[0]) and w['lines'] == int(args.values[1])
                   for w in json.loads(current)):
                print(current)
                return
            time.sleep(.1)
        raise TimeoutError('Kitty did not reach the requested terminal dimensions')
    elif args.action == 'wait':
        target = args.values[0]
        deadline = time.monotonic() + args.timeout
        while time.monotonic() < deadline:
            if target in control(root, 'get-text', '--extent', 'screen'):
                record(root, 'wait', text=target)
                return
            time.sleep(.1)
        snapshot(root, 'timeout-' + str(int(time.time())))
        raise TimeoutError(f'Visible text did not contain {target!r}')
    elif args.action == 'stop':
        stop(root)


if __name__ == '__main__':
    main()
