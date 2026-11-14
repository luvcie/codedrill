package typingsound

// PackInfo identifies a community TETR.IO+ sound pack. The catalog stores only
// metadata: audio stays on the community host and is downloaded on demand.
type PackInfo struct {
	ID     string
	Name   string
	Author string
	URL    string
}

// SilentPackID is the default selection and plays nothing.
const SilentPackID = "silent"

// PrototypePackID selects the generated diagnostic tones.
const PrototypePackID = "prototype"

// TETRIODatabase is the gallery the TETR.IO+ packs come from.
const TETRIODatabase = "https://you.have.fail/tetrioplus/"

// Catalog is the set of TETR.IO+ packs from transcendence's public catalog.
// Names and links identify the community work; no audio is bundled.
var Catalog = []PackInfo{
	{ID: "sfx-Ayx-GameCube_Controller_SFX", Name: "GameCube Controller SFX", Author: "Ayx", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Ayx/GameCube_Controller_SFX.zip.tpse"},
	{ID: "sfx-Ayx-Wii_Sports_Resort_SFX", Name: "Wii Sports Resort SFX", Author: "Ayx", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Ayx/Wii_Sports_Resort_SFX.zip.tpse"},
	{ID: "sfx-Azura-terriblememesfx", Name: "Terrible Meme Sounds", Author: "Azura", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Azura/terriblememesfxver1.zip.tpse"},
	{ID: "sfx-Ferln4-Bejeweled_ish_SFX_pack", Name: "Bejeweled-ish SFX", Author: "Ferln4", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Ferln4/Bejeweled_ish_SFX_pack.zip.tpse"},
	{ID: "sfx-Ferln4-Minecraft_SFX", Name: "Minecraft-y Soundpack", Author: "Ferln4", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Ferln4/Minecraft_SFX.zip.tpse"},
	{ID: "sfx-Ferln4-Misamino_Soundpack", Name: "Misamino / Tetris Online Poland SFX", Author: "Ferln4", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Ferln4/Misamino_Soundpack.zip.tpse"},
	{ID: "sfx-Ferln4-Touhou_SFX", Name: "Touhou SFX", Author: "Ferln4", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Ferln4/Touhou_SFX.zip.tpse"},
	{ID: "sfx-FiFiZ-tetrazz_pulse_x_sfxpack", Name: "Tetrazz SFX PACK - Pulse X", Author: "FiFiZ", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/FiFiZ/tetrazz_pulse_x_sfxpack.zip.tpse"},
	{ID: "sfx-Fulter-Another_SFX_Randoms", Name: "Another SFX Randoms", Author: "Fulter", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Fulter/Another_SFX_Randoms.zip.tpse"},
	{ID: "sfx-Fulter-Apotris_Sound_Tetrio", Name: "Apotris (some SFX)", Author: "Fulter", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Fulter/Apotris_Sound_Tetrio.zip.tpse"},
	{ID: "sfx-Fulter-The_SNESTRIS_sound_pack", Name: "The SNESTRIS Sound Pack", Author: "Fulter", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Fulter/The_SNESTRIS_sound_pack.zip.tpse"},
	{ID: "sfx-Huetoast-21st_century_humor", Name: "21st century humor", Author: "Huetoast", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Huetoast/21st_century_humor.zip.tpse"},
	{ID: "sfx-Itsmega-BejeweledSR", Name: "Bejeweled Soundpack Rearranged", Author: "Itsmega", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Itsmega/BejeweledSR.zip.tpse"},
	{ID: "sfx-Manana-DOOM", Name: "DOOM", Author: "Manana", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Manana/DOOM.zip.tpse"},
	{ID: "sfx-Manana-ST3", Name: "SUPER Tetris 3", Author: "Manana", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Manana/ST3.zip.tpse"},
	{ID: "sfx-NikoAnimation-Super_Stardust_HD_SFX_WIP_v1_0", Name: "Super Stardust HD Soundpack (WIP v1.0)", Author: "NikoAnimation", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/NikoAnimation/Super_Stardust_HD_SFX_WIP_v1_0.zip.tpse"},
	{ID: "sfx-NikoAnimation-Undertale_Soundpack_v1_0_3", Name: "Undertale soundpack (WIP v1.0.3)", Author: "NikoAnimation", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/NikoAnimation/Undertale_Soundpack_v1_0_3.zip.tpse"},
	{ID: "sfx-Octoling-pizzatower_soundexport", Name: "Pizza Tower Soundpack V1", Author: "Octoling", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Octoling/pizzatower_soundexport.zip.tpse"},
	{ID: "sfx-penguin_colonel-RainWorldSFX", Name: "Rain World SFX Pack", Author: "penguin_colonel", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/penguin_colonel/RainWorldSFX.zip.tpse"},
	{ID: "sfx-risograph-botwtetriose", Name: "Breath of the Wild SFX Pack", Author: "risograph", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/risograph/botwtetriose.zip.tpse"},
	{ID: "sfx-sayketsu-bookworm_sfx_beefstew", Name: "Bookworm Adventures Soundpack", Author: "sayketsu", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/sayketsu/bookworm_sfx_beefstew.zip.tpse"},
	{ID: "sfx-sourTaste000-se", Name: "Yotipo's SE", Author: "sourTaste000", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/sourTaste000/se.zip.tpse"},
	{ID: "sfx-Spectremint-tetris_ps3sfx", Name: "Tetris PS3 SFX", Author: "Spectremint", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Spectremint/tetris_ps3sfx.zip.tpse"},
	{ID: "sfx-spep_hsrih-tgm3_ti_se_pack_v_idk_this_was_done_in_two_da", Name: "Tetris The Grandmaster 3 se pack", Author: "spep_hsrih", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/spep_hsrih/tgm3_ti_se_pack_v_idk_this_was_done_in_two_da.zip.tpse"},
	{ID: "sfx-Starcat_JP-8Bit_Puzzler", Name: "8Bit Puzzler", Author: "Starcat_JP", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Starcat_JP/8Bit_Puzzler.zip.tpse"},
	{ID: "sfx-Starcat_JP-Future_Wave_Sound_effect_dark_edit", Name: "Future Wave \"dark edit\"", Author: "Starcat_JP", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Starcat_JP/Future_Wave_Sound_effect_dark_edit.zip.tpse"},
	{ID: "sfx-Starcat_JP-Future_Wave_Sound_effect", Name: "Future Wave", Author: "Starcat_JP", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Starcat_JP/Future_Wave_Sound_effect.zip.tpse"},
	{ID: "sfx-Tetrzite-PAC_MAN_Soundpack", Name: "PAC-MAN Soundpack", Author: "Tetrzite", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Tetrzite/PAC_MAN_Soundpack.zip.tpse"},
	{ID: "sfx-Walpang-Dubstep_SFX_Pack", Name: "Dubstep SFX Pack", Author: "Walpang", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Walpang/Dubstep_SFX_Pack.zip.tpse"},
}

// FindPack looks a pack up by ID. The prototype and silent selections are not
// community packs and are not in Catalog.
func FindPack(id string) (PackInfo, bool) {
	for _, pack := range Catalog {
		if pack.ID == id {
			return pack, true
		}
	}
	return PackInfo{}, false
}

// PackLabel returns a display name for a stored pack selection.
func PackLabel(id string) string {
	switch id {
	case "", SilentPackID:
		return "Silent"
	case PrototypePackID:
		return "Generated tones (prototype)"
	}
	if pack, ok := FindPack(id); ok {
		return pack.Name
	}
	return "Unknown pack (" + id + ")"
}
