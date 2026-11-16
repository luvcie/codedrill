// Package music provides optional, locally cached drill music. No audio is bundled.
package music

const (
	SprintZen   = "sprint-zen"
	TetraLeague = "tetra-league"
	Holidays    = "holidays-2023"
	Local       = "local"
)

type Track struct {
	ID, Name, Artist, URL string
	Size                  int64
}

// HURT RECORD entries follow the source sheet; sizes are HTTP lengths checked in November 2026.
var Calm = []Track{
	{"kvx-016", "Piercing Wind", "KVK", "https://www.hurtrecord.com/bgm/16/honemi-ni-shimiiru-karasukaze.mp3", 3497357},
	{"chk-016", "Inorimichite", "Chika", "https://www.hurtrecord.com/bgm/16/inorimichite.mp3", 4106465},
	{"chk-038", "Wind Trail", "Chika", "https://www.hurtrecord.com/bgm/38/kaze-no-sanpomichi.mp3", 4557504},
	{"tht-037", "Muscat and White Dishes", "Takahashi Takashi", "https://www.hurtrecord.com/bgm/37/muscat-to-shiroi-osara.mp3", 4805182},
	{"tht-042", "Summer Sky and Homework", "Takahashi Takashi", "https://www.hurtrecord.com/bgm/42/natsuzora-to-syukudai.mp3", 3669256},
	{"chk-055", "Akindo", "Chika", "https://www.hurtrecord.com/bgm/55/akindo.mp3", 3103864},
	{"abm-047", "Rainbow of the Night", "Aiba Makoto", "https://www.hurtrecord.com/bgm/47/yoru-no-niji.mp3", 3553254},
	{"kmk-053", "Step on the Scarlet Soil", "Kamoking", "https://www.hurtrecord.com/bgm/53/akai-tsuchi-wo-funde.mp3", 3412260},
	{"jgm-044", "Hanging Out in Tokyo", "Meesan", "https://www.hurtrecord.com/bgm/44/burari-tokyo.mp3", 3837435},
	{"abm-048", "Prism", "Aiba Makoto", "https://www.hurtrecord.com/bgm/48/prism.mp3", 2980799},
	{"sdm-016", "Winter Satellite", "Sudo Mikaduki", "https://www.hurtrecord.com/bgm/16/fuyu-no-jinkoueisei.mp3", 4446298},
	{"yos-016", "First Snow", "Yoshi", "https://www.hurtrecord.com/bgm/16/hatsuyuki.mp3", 3607230},
	{"chk-047", "Main Street", "Chika", "https://www.hurtrecord.com/bgm/47/kansen-gairo.mp3", 4258781},
	{"omg-019", "Twenty-First Century People", "Omegane", "https://www.hurtrecord.com/bgm/19/21seiki-no-hitobito.mp3", 3488692},
	{"omg-016", "Waiting for Spring to Come", "Omegane", "https://www.hurtrecord.com/bgm/16/haru-wo-machinagara.mp3", 4046226},
	{"nbh-022", "Go Go Go Summer", "Nobuhamu", "https://www.hurtrecord.com/bgm/22/go-go-go-summer.mp3", 2841538},
	{"nhr-040", "Lonely Journey", "Naoki Hirai", "https://www.hurtrecord.com/bgm/40/sasurai-no-hitoritabi.mp3", 3840133},
	{"chk-004", "Young Leaves", "Chika", "https://www.hurtrecord.com/bgm/04/wakana.mp3", 4086412},
	{"omg-051", "Confession", "Omegane", "https://www.hurtrecord.com/bgm/51/zange-no-ma.mp3", 2580983},
	{"ttm-055", "Amazing Everyday", "Tsutomu", "https://www.hurtrecord.com/bgm/55/subarashii-nichijou.mp3", 4043884},
	{"chk-009", "Asphalt", "Chika", "https://www.hurtrecord.com/bgm/09/asphalt.mp3", 2497368},
	{"chk-048", "By the Sunlit Window", "Chika", "https://www.hurtrecord.com/bgm/48/madobe-no-hidamari.mp3", 2831656},
	{"chk-053", "Origin", "Chika", "https://www.hurtrecord.com/bgm/53/minamoto.mp3", 6688321},
	{"chk-041", "Cherry Blossom Season", "Chika", "https://www.hurtrecord.com/bgm/41/sora-no-sakura.mp3", 6569854},
	{"chk-021", "Raindrops", "Chika", "https://www.hurtrecord.com/bgm/21/suiu.mp3", 2945503},
	{"chk-014", "Entrance Wreath", "Chika", "https://www.hurtrecord.com/bgm/14/freshherb-wreath-wo-genkan-ni.mp3", 2684305},
}

var Battle = []Track{
	{"kmk-033", "Backwater", "Kamoking", "https://www.hurtrecord.com/bgm/33/back-water.mp3", 3237928},
	{"kmk-024", "Burning Heart", "Kamoking", "https://www.hurtrecord.com/bgm/24/burning-heart.mp3", 2947654},
	{"kmk-038", "Storm Spirit", "Kamoking", "https://www.hurtrecord.com/bgm/38/hayate-no-sei.mp3", 3428572},
	{"kmk-047", "Ice Eyes", "Kamoking", "https://www.hurtrecord.com/bgm/47/ice-eyes.mp3", 3101695},
	{"tmk-060", "The Time is Now", "Tomoki", "https://www.hurtrecord.com/bgm/60/ima-koso.mp3", 3132375},
	{"mkk-033", "Risky Area", "Mikiya Komaba", "https://www.hurtrecord.com/bgm/33/risky-area.mp3", 6847064},
	{"kmk-036", "Over The Horizon", "Kamoking", "https://www.hurtrecord.com/bgm/36/chiheisen-wo-koete.mp3", 2504317},
	{"kmk-017", "Burning Spirit, Awakening Soul", "Kamoking", "https://www.hurtrecord.com/bgm/17/moyase-toushi-yobisamase-tamashii.mp3", 2741890},
	{"kmk-032", "Maze of the Abyss", "Kamoking", "https://www.hurtrecord.com/bgm/32/naraku-heno-abyssmaze.mp3", 3612504},
	{"kmk-004", "Samurai Sword", "Kamoking", "https://www.hurtrecord.com/bgm/04/samurai-sword.mp3", 3186126},
	{"kmk-041", "Super Machine Soul", "Kamoking", "https://www.hurtrecord.com/bgm/41/super-machine-soul.mp3", 3640987},
	{"kmk-039", "Universe 5239", "Kamoking", "https://www.hurtrecord.com/bgm/39/uchuu-5239.mp3", 3029886},
	{"kmk-051", "Ultra Super Heroes", "Kamoking", "https://www.hurtrecord.com/bgm/51/ultra-super-heros.mp3", 3128949},
}

var HolidayTracks = []Track{
	{ID: "christmas-city", Name: "クリスマス都市 (Christmas City)", Artist: "Dr Ocelot", Size: 3252061},
	{ID: "carol-of-the-blocks", Name: "Carol of the Blocks", Artist: "Dr Ocelot", Size: 4621841},
}

func Tracks(source string) []Track {
	switch source {
	case SprintZen:
		return Calm
	case TetraLeague:
		return Battle
	case Holidays:
		return HolidayTracks
	}
	return nil
}
