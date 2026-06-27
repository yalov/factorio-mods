local function sensor(player)
    local settings_coord = player.mod_settings["statsgui-cd-coordinates"].value
    local settings_dist = player.mod_settings["statsgui-cd-distance"].value

    if not (settings_coord or settings_dist) or player.character == nil then
        return {"", "", ""}
    end

    local gui_value = ""

    -- coordinates display
    if (settings_coord) then
        gui_value = string.format("X = %.1f, Y = %.1f", player.position.x, player.position.y)
    end

    if (settings_coord and settings_dist) then
        gui_value = gui_value .. ", "
    end

    -- distance display
    if (settings_dist) then
        -- get the distance between the player's position and the set position in the mod settings
        local dx = player.position.x - player.mod_settings["statsgui-cd-position-x"].value
        local dy = player.position.y - player.mod_settings["statsgui-cd-position-y"].value
        local distance = (dx ^ 2 + dy ^ 2) ^ 0.5

        if (settings_coord) then
            gui_value = gui_value .. string.format("D = %.1f", distance)
        else
            gui_value = string.format("Distance = %.1f", distance)
        end
    end

    return {"", "", gui_value}

end

local function register_sensor()
    -- always call the `version` function first to avoid crashes if the interface changes in the future
    if script.active_mods["StatsGui"] and remote.call("StatsGui", "version") == 1 then
        remote.call("StatsGui", "add_sensor", "StatsGui-CoordinatesDistance", "cd_sensor")
    end
end

script.on_init(function()
    register_sensor()
end)

script.on_load(function()
    register_sensor()
end)

remote.add_interface("StatsGui-CoordinatesDistance", {
    cd_sensor = sensor
})
