package typingsound

// PackFormat describes how a pack stores its audio.
type PackFormat uint8

const (
	// FormatTPSE is a TETR.IO+ JSON export with one recording and an atlas.
	FormatTPSE PackFormat = iota
	// FormatZIP is a native TETR.IO sound-pack ZIP with one audio file per effect.
	FormatZIP
)

// PackInfo identifies a community TETR.IO+ sound pack. The catalog stores only
// metadata: audio stays on the community host and is downloaded on demand.
type PackInfo struct {
	ID     string
	Name   string
	Author string
	URL    string
	Format PackFormat
	Source string
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

	// Newer catalog entries, including legacy YHF uploads whose type is not
	// Sound Effects and variants that are not separate gallery cards.
	{ID: "sfx-25Pi25-soundpack", Name: "Enhanced Drumkit", Author: "25Pi25", URL: "https://you.have.fail/tetrioplus/data/content/sfx/25Pi25/soundpack.zip", Format: FormatZIP},
	{ID: "sfx-8bJackie-Tetrio_retro_pack", Name: "Tetrio_retro_pack", Author: "8bJackie", URL: "https://you.have.fail/tetrioplus/data/content/sfx/8bJackie/Tetrio_retro_pack.zip", Format: FormatZIP},
	{ID: "sfx-aznguy.mp4-tetroblox_sfx_zip_idk", Name: "Tetroblox SFX", Author: "aznguy.mp4", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/aznguy.mp4/tetroblox_sfx_zip_idk.zip.tpse"},
	{ID: "sfx-Ayx-MOTHER23", Name: "MOTHER 2+3 SFX", Author: "Ayx", URL: "https://you.have.fail/tetrioplus/data/content/sfx/Ayx/MOTHER23.zip", Format: FormatZIP},
	{ID: "sfx-Deathic-TheEmeraldPalace", Name: "The Emerald Palace SFX", Author: "Deathic", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Deathic/TheEmeraldPalace.zip.tpse"},
	{ID: "sfx-Manana-Charming_Jirou", Name: "Charming_Jirou_Chang", Author: "Manana", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Manana/Charming_Jirou.zip.tpse"},
	{ID: "sfx-nagi_desuuu-back2boom", Name: "Back2Boom", Author: "nagi_desuuu", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/nagi_desuuu/back2boom.zip.tpse"},
	{ID: "sfx-Neeko-TETR.IO_Friends_Voicelines_(Very_Real)", Name: "TETR.IO Friends Voicelines (Very Real)", Author: "Neeko", URL: "https://you.have.fail/tetrioplus/data/content/sfx/Neeko/TETR.IO_Friends_Voicelines_(Very_Real).zip", Format: FormatZIP},
	{ID: "sfx-Starcat_JP-Future_Wave_qp2_sfx", Name: "Future Wave QP2 SFX add-ons", Author: "Starcat_JP", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/sfx/Starcat_JP/Future_Wave_qp2_sfx.zip.tpse"},
	{ID: "sfx-Wukov-Super_Tetrio_World_Sound_Pack", Name: "Super Tetrio World SoundPack", Author: "Wukov", URL: "https://you.have.fail/tetrioplus/data/content/sfx/Wukov/Super_Tetrio_World_Sound_Pack.zip", Format: FormatZIP},
	{ID: "sfx-AltAccountGirl-tb_sfx", Name: "tb_sfx", Author: "AltAccountGirl", URL: "https://you.have.fail/tetrioplus/data/content/sfx/AltAccountGirl/tb_sfx.tpse"},
	{ID: "sfx-AltAccountGirl-tb_sfx_reverb", Name: "tb_sfx_reverb", Author: "AltAccountGirl", URL: "https://you.have.fail/tetrioplus/data/content/sfx/AltAccountGirl/tb_sfx_reverb.tpse"},
	{ID: "sfx-AltAccountGirl-tetra_online_sfx", Name: "Tetra Online", Author: "AltAccountGirl", URL: "https://you.have.fail/tetrioplus/data/content/sfx/AltAccountGirl/tetra_online_sfx.tpse"},
	{ID: "sfx-idkwhyiamhereandwhoami-NES", Name: "NES", Author: "idkwhyiamhereandwhoami", URL: "https://you.have.fail/tetrioplus/data/content/sfx/idkwhyiamhereandwhoami/NES.tpse"},
	{ID: "sfx-Neeko-betterPPTSFX", Name: "betterPPTSFX", Author: "Neeko", URL: "https://you.have.fail/tetrioplus/data/content/sfx/Neeko/betterPPTSFX.tpse"},
	{ID: "sfx-NOT_A_ROBOT-tgm3", Name: "TGM3 SFX", Author: "NOT_A_ROBOT", URL: "https://you.have.fail/tetrioplus/data/content/sfx/NOT_A_ROBOT/tgm3.tpse"},
	{ID: "sfx-nuwop-four_tris_sound_effects", Name: "Four-Tris Sound Effects", Author: "nuwop", URL: "https://you.have.fail/tetrioplus/data/content/sfx/nuwop/four-tris_sound_effects.tpse"},
	{ID: "sfx-nuwop-jstris", Name: "Jstris Nullpomino SFX (with osu menu sounds)", Author: "nuwop", URL: "https://you.have.fail/tetrioplus/data/content/sfx/nuwop/jstris.tpse"},
	{ID: "sfx-nuwop-nullpomino_classic", Name: "Classic NullpoMino Sound Effects", Author: "nuwop", URL: "https://you.have.fail/tetrioplus/data/content/sfx/nuwop/nullpomino-classic.tpse"},
	{ID: "sfx-nuwop-topsfx", Name: "Tetris Online Poland - Sound Effects (Edited)", Author: "nuwop", URL: "https://you.have.fail/tetrioplus/data/content/sfx/nuwop/topsfx.tpse"},
	{ID: "sfx-nuwop-Tetroblox_x_Super_Mario_World_Sound_Effects", Name: "Tetroblox & Super Mario World Sound Effects", Author: "nuwop", URL: "https://you.have.fail/tetrioplus/data/content/sfx/nuwop/Tetroblox_x_Super_Mario_World_Sound_Effects.tpse"},
	{ID: "sfx-ZaptorZap-tetra_legends_se_pack", Name: "tetra-legends-se-pack", Author: "ZaptorZap", URL: "https://you.have.fail/tetrioplus/data/content/sfx/ZaptorZap/tetra-legends-se-pack.tpse"},
	{ID: "sfx-ZaptorZap-tetrio_crunch_pack", Name: "tetrio-crunch-pack", Author: "ZaptorZap", URL: "https://you.have.fail/tetrioplus/data/content/sfx/ZaptorZap/tetrio-crunch-pack.tpse"},
	{ID: "sfx-ZaptorZap-tetris_friends_sepack", Name: "tetris-friends-sepack", Author: "ZaptorZap", URL: "https://you.have.fail/tetrioplus/data/content/sfx/ZaptorZap/tetris-friends-sepack.tpse"},
	{ID: "sfx-ZaptorZap-tetris99_tetrio_se_pack", Name: "tetris99-tetrio-se-pack", Author: "ZaptorZap", URL: "https://you.have.fail/tetrioplus/data/content/sfx/ZaptorZap/tetris99-tetrio-se-pack.tpse"},
	{ID: "sfx-ZaptorZap-tetris_party_deluxe_minipack", Name: "tetris_party_deluxe_minipack", Author: "ZaptorZap", URL: "https://you.have.fail/tetrioplus/data/content/sfx/ZaptorZap/tetris_party_deluxe_minipack.tpse"},
	{ID: "unknown-AviatorSam-HL2_sfx_pack_final", Name: "Half-Life 2 SFX Pack", Author: "AviatorSam", URL: "https://you.have.fail/tetrioplus/data/content/unknown/AviatorSam/HL2_sfx_pack_final.zip", Format: FormatZIP},
	{ID: "unknown-garbaggy-sfx_zip", Name: "Tetris1989", Author: "garbaggy", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/unknown/garbaggy/sfx_zip.zip.tpse"},
	{ID: "unknown-Creta_Park-tetr_ds_lite", Name: "Tetr.ds lite (microplus compatible)", Author: "Creta Park", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/unknown/Creta_Park/tetr_ds_lite.tpse.tpse"},
	{ID: "unknown-Starcat_JP-Starcats_Cute_Skin_Pack", Name: "Starcat's Cute Skin Pack!", Author: "Starcat_JP", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/unknown/Starcat_JP/Starcats_Cute_Skin_Pack.tpse.tpse"},
	{ID: "unknown-Starcat_JP-Tetrio_Potion_Brewing_Pack", Name: "Potion Brewing Pack", Author: "Starcat_JP", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/unknown/Starcat_JP/Tetrio_Potion_Brewing_Pack.tpse.tpse"},
	{ID: "unknown-_conc1erge_-Another_TF_Battle_2P_SFX_Pack", Name: "Another TF Battle 2P SFX Pack", Author: "conc1erge", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/unknown/_conc1erge_/Another_TF_Battle_2P_SFX_Pack.tpse.tpse"},
	{ID: "unknown-Ferln4-pitchshifted_1", Name: "Lower Pitch Default", Author: "Ferln4", URL: "https://you.have.fail/tetrioplus/data/tpsefiles/unknown/Ferln4/pitchshifted_1.tpse.tpse"},
	{ID: "sfx-TetrisBattleSFX", Name: "Tetris Battle SFX", Author: "Unknown", URL: "https://drive.usercontent.google.com/download?id=11F9yErJ5Pqgk8xmQNH_DcfnXa2NDsRAI&export=download&confirm=t", Format: FormatZIP, Source: "https://drive.google.com/file/d/11F9yErJ5Pqgk8xmQNH_DcfnXa2NDsRAI/view"},
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
