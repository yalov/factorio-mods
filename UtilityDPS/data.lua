
data:extend {
    {
        type = "shortcut",
        name = "dps-gui-shortcut",    -- Unique name
        action = "lua",               -- Allows Lua script handling
        localised_name = "DPS GUI" , -- Display name (can use locale strings)
        icon = "__UtilityDPS__/graphics/dps_icon_56.png",
        icon_size = 56,
        small_icon = "__UtilityDPS__/graphics/dps_icon_24.png",
        small_icon_size = 24,
        toggleable = true,                                -- Set to false if it's just a one-time click
        associated_control_input = "dps-gui-toggle-input" -- Links to a custom input (defined below)
    },
    {
        type = "custom-input",
        name = "dps-gui-toggle-input", -- Matches the shortcut's associated_control_input
        key_sequence = "",             -- Empty means no default hotkey; players can bind one
        action = "lua"
    }
}


-- Sprites
local toolbar_icons = "__UtilityDPS__/graphics/toolbar-icons.png"

data:extend {
    {
        type = "sprite",
        name = "dps_play",
        filename = "__UtilityDPS__/graphics/dps_play.png",
        position = { 0, 0 },
        size = 32,
        flags = { "icon" },
    },
    {
        type = "sprite",
        name = "dps_pause",
        filename = "__UtilityDPS__/graphics/dps_pause.png",
        position = { 0, 0 },
        size = 32,
        flags = { "icon" },
    },
        {
        type = "sprite",
        name = "dps_eraser",
        filename = "__UtilityDPS__/graphics/dps_eraser.png",
        position = { 0, 0 },
        size = 32,
        flags = { "icon" },
    },
    {
        type = "sprite",
        name = "dps_log_white",
        filename = "__UtilityDPS__/graphics/dps_log_white.png",
        position = { 0, 0 },
        size = 32,
        flags = { "icon" },
    },

    {
        type = "sprite",
        name = "dps_log_black",
        filename = "__UtilityDPS__/graphics/dps_log_black.png",
        position = { 0, 0 },
        size = 32,
        flags = { "icon" },
    },

    {
        type = "sprite",
        name = "dps_settings_white",
        filename = toolbar_icons,
        position = { 32, 0 },
        size = 32,
        flags = { "icon" },
    },
    {
        type = "sprite",
        name = "dps_settings_black",
        filename = toolbar_icons,
        position = { 0, 0 },
        size = 32,
        flags = { "icon" },
    },
    {
        type = "sprite",
        name = "dps_pin_white",
        filename = toolbar_icons,
        position = { 32, 32 },
        size = 32,
        flags = { "icon" },
    },
    {
        type = "sprite",
        name = "dps_pin_black",
        filename = toolbar_icons,
        position = { 0, 32 },
        size = 32,
        flags = { "icon" },
    },

}
