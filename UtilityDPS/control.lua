-- TODO: damage before res / after res show at the same time?

local DAMAGE_TYPE_COLORS = {
    physical  = "#b2b2b2", -- light gray (like item icons)
    explosion = "#ffb347", -- orange-yellow (explosives)
    fire      = "#ff5c33", -- bright red-orange (fire/flame)
    laser     = "#FFA08F", -- "#e43fff", -- magenta (laser)
    electric  = "#a8f8ff", -- cyan (electric beams)
    acid      = "#7fff00",  -- bright green (acid spitters)
    poison    = "#05A1AF"
}

-- constants
local LOG_CAPACITY = 200          -- max log lines kept per player
-- local STALE_EVICT_TICKS = 60 * 60 -- (future) ticks of inactivity before pruning (already inline elsewhere)

local CAUSE_ITEMS = {"character", "player force", "player+0 force", "enemy", "any"}
local TARGET_ITEMS = {"enemy", "not ally", "any"}



-- Allowed discrete update interval tick values (replaces free slider)
local ALLOWED_INTERVAL_TICKS = {5, 10, 20, 30, 60}
local INTERVAL_ITEMS = {"5","10","20","30","60"}
-- Discrete window seconds values
local ALLOWED_WINDOW_SECONDS = {1, 2, 5, 10, 20, 30, 60}
local WINDOW_ITEMS = {"1","2","5","10","20","30","60"}

local function interval_index_for(value)
    -- find exact match
    for i,v in ipairs(ALLOWED_INTERVAL_TICKS) do
        if v == value then return i end
    end
    -- fallback
    return ALLOWED_INTERVAL_TICKS[#ALLOWED_INTERVAL_TICKS]
end

local function window_index_for(value)
    for i,v in ipairs(ALLOWED_WINDOW_SECONDS) do
        if v == value then return i end
    end
    -- choose nearest (simple linear scan, small set)
    local best_i, best_diff = 1, math.huge
    for i,v in ipairs(ALLOWED_WINDOW_SECONDS) do
        local d = math.abs(v - value)
        if d < best_diff then best_diff = d; best_i = i end
    end
    return best_i
end



-- small helper to validate GUI element
local function valid(e) return (e and e.valid) end

-- get player and player's data; returns nil,nil if any not found
local function get_player_and_data(player_index)
    local player = game.get_player(player_index)
    if not player then return nil,nil end
    local pdata = storage.players and storage.players[player.index]
    if not pdata then return nil,nil end
    return player, pdata
end



local function initialize_storage()
    -- Ensure storage table exists
    storage = storage or {}
    -- Initialize mod-specific data per player
    storage.players = storage.players or {}
    storage.active_players = storage.active_players or {} -- players with open DPS window

    storage.dps_nth_tick_registered = storage.dps_nth_tick_registered or nil
    storage.dps_damage_handler_registered = storage.dps_damage_handler_registered or false
    storage.update_interval_ticks = storage.update_interval_ticks or 10
    
end


local function prepare_player_data(player)
	local data = {}
	data.frame = nil
	data.text_box = nil
    data.text_box_flow = nil
	data.content = ""
	data.location_x = 0
	data.location_y = 0
	data.has_location = false
	data.settings_visible = true
	data.log_visible = true
	data.dps_window_seconds = 30
	data.dps_window_open = false

    -- cached filter indices (GUI widgets mirror these)
    data.cause_selected_index = 1
    data.target_selected_index = 1
    data.all_surfaces = false

	storage.players[player.index] = data
end

local function configuration_changed()
    initialize_storage()
    -- Reset per-player damage data (now stored directly on player data)
    for _, pdata in pairs(storage.players) do
        if pdata then
            pdata.damage_by_type = nil
        end
    end
    -- Ensure player data exists for all current players (handles adding mod to existing save)
    for _, player in pairs(game.players) do
        if valid(player) then
            if not (storage.players and storage.players[player.index]) then
                prepare_player_data(player)
            end
        end
    end
end

-- when the mod is first added or a new game is created.
script.on_init(function()
    initialize_storage()
    -- Create data for all existing players (new game or added to existing save)
    for _, player in pairs(game.players) do
        if valid(player) then
            prepare_player_data(player)
        end
    end
end)

-- when the mod is updated, added, removed, or startup settings change
script.on_configuration_changed(function(event)
    configuration_changed()
end)




local function player_created(event)
	prepare_player_data(game.get_player(event.player_index))
end

local function player_removed(event)
    -- Keep indices stable; don't reindex the table
        storage.players[event.player_index] = nil
        if storage.active_players then
            storage.active_players[event.player_index] = nil
        end
    end

local function get_player_data(player)
    if not player then return nil end
    storage.players = storage.players or {}
    local pdata = storage.players[player.index]
    if not pdata and player.valid then
        -- Lazily create player data if missing (e.g., mod added mid-save before shortcut used)
        prepare_player_data(player)
        pdata = storage.players[player.index]
    end
    return pdata
end

local function gui_location_changed(event)
	if event.element.name=="frame" then
		local data = get_player_data(game.get_player(event.player_index))
		data.location_x = event.element.location.x
		data.location_y = event.element.location.y
		data.has_location = true
	end
end


script.on_event(defines.events.on_player_created, player_created)
script.on_event(defines.events.on_player_removed, player_removed)
script.on_event(defines.events.on_gui_location_changed, gui_location_changed)


local function set_dps_gui(player, text)
    if not valid(player) then return end
    local data = get_player_data(player)
    if data and valid(data.dps_label) then
        data.dps_label.caption = text
    end
end

-- Utility function to set or update the DPS label above the character
local function set_dps_screen_label(player, text, lines_count)
    if not player or not player.valid then return end
    local label = player.gui.screen.dps_screen_label
    if label and label.valid then
        label.caption = text
    else
        label = player.gui.screen.add{
            type = "label",
            name = "dps_screen_label",
            caption = text
        }
    end

    label.style.font = "default"
    label.style.single_line = false
    label.style.maximal_width = 300
    label.style.horizontal_align = "left"

    local res = player.display_resolution
    local center_x = math.floor(res.width / 2)
    local center_y = math.floor(res.height / 2)

    local label_height = lines_count * 20
    label.location = {center_x, center_y - 100 - label_height} -- 100 pixels above center
end

local function on_nth_tick_handler(event)
    local minute_ticks = 60 * 60
    if not storage.active_players then return end
    for player_index, _ in pairs(storage.active_players) do
        local player, data = get_player_and_data(player_index)
        if not player or not data or not data.dps_window_open or not data.damage_by_type then goto continue end

        local window_ticks = data.dps_window_seconds * 60
        local buckets_max = math.max(1, math.floor(window_ticks / storage.update_interval_ticks))

        for damage_type, type_data in pairs(data.damage_by_type) do
            if type_data.pending_damage == nil then
                -- initialize new schema lazily
                type_data.pending_damage = 0
                type_data.window_sums = {}
                type_data.window_sums_start = 1 -- ring buffer logical start
                type_data.window_sums_tail = 0   -- explicit tail index (0 means empty)
                type_data.window_total = 0
            end

            local pdmg = type_data.pending_damage
            type_data.pending_damage = 0
            local ws = type_data.window_sums
            local start_i = type_data.window_sums_start or 1

            -- assume window_sums_tail always present (migration logic removed per request)

            -- append new bucket at explicit tail index (avoid # on sparse array)
            local tail = type_data.window_sums_tail + 1
            ws[tail] = pdmg
            type_data.window_sums_tail = tail
            type_data.window_total = type_data.window_total + pdmg

            -- compute logical length (may be zero if empty)
            local logical_len = tail - start_i + 1

            -- trim excess buckets by advancing start pointer
            while logical_len > buckets_max do
                local old = ws[start_i]
                if old then
                    type_data.window_total = type_data.window_total - old
                    ws[start_i] = nil -- mark removable; may compact later
                end
                start_i = start_i + 1
                logical_len = tail - start_i + 1
            end
            type_data.window_sums_start = start_i

            -- occasional compaction: rebuild contiguous array when head far ahead
            if start_i > 64 and start_i > (logical_len * 2) then
                local new = {}
                local n = 0
                for i = start_i, tail do
                    local v = ws[i]
                    if v then
                        n = n + 1
                        new[n] = v
                    end
                end
                type_data.window_sums = new
                type_data.window_sums_start = 1
                type_data.window_sums_tail = n
                ws = new
                start_i = 1
                tail = n
                logical_len = n
                --log( string.format("ws: compacted n=%d", n) )
            end

            -- log( string.format("ws: start=%d tail=%d llen=%d pdmg=%d total=%d %s", 
            --  start_i, type_data.window_sums_tail,  logical_len, pdmg, type_data.window_total, serpent.line(ws)) )

            if event.tick - (type_data.last_seen or event.tick) > minute_ticks then
                data.damage_by_type[damage_type] = nil
            end
        end

        local type_dps_strings = {}
        for damage_type, type_data in pairs(data.damage_by_type) do
            local total = type_data.window_total or 0
            local dps = total / data.dps_window_seconds
            local color = DAMAGE_TYPE_COLORS[damage_type] or "#ffffff"
            type_dps_strings[#type_dps_strings+1] = string.format("%s: [font=default-bold][color=%s]%8d[/color][/font]", damage_type, color, dps)
        end

        if #type_dps_strings > 0 then
            set_dps_gui(player, table.concat(type_dps_strings, "\n"))
        else
            set_dps_gui(player, "No damage")
        end
        ::continue::
    end
end


-- Add a message to the player's GUI
-- internal helper to rebuild the visible log text efficiently
local function rebuild_log_text(data)
    if not valid(data.text_box) or not data.messages then return end
    local start_index = data.messages_start or 1
    local msgs = data.messages
    data.messages_buffer = data.messages_buffer or {}
    local buffer = data.messages_buffer
    -- clear buffer
    for i = 1, #buffer do buffer[i] = nil end
    for i = start_index, #msgs do
        local line = msgs[i]
        if line then
            buffer[#buffer+1] = line
        end
    end
    data.text_box.text = table.concat(buffer, "\n")
end


local function text_box_create(data)
    
    data.text_box_flow = data.horizontal_pane.add({type="flow", name="text_box_flow", direction="vertical"})
    local text_box_buttons_flow = data.text_box_flow.add({type="flow", name="text_box_buttons_flow", direction="horizontal"})

    data.logging_paused = data.logging_paused or false
    data.log_toggle_btn = text_box_buttons_flow.add{
        type = "sprite-button",
        name = "log_toggle_btn",
        sprite = data.logging_paused and "dps_pause" or "dps_play",
        style = "frame_action_button",
        tooltip = data.logging_paused and "Paused Logging" or "Running Logging"
    }

    -- clear button
    data.log_clear_btn = text_box_buttons_flow.add{
       type = "sprite-button",
       name = "log_clear_btn",
       sprite = "dps_eraser",
       style = "frame_action_button",
       tooltip = "Clear Log"
    }

    data.text_box = data.text_box_flow.add({type="text-box", name="dps_log"})
    data.text_box.word_wrap = false
    data.text_box.read_only = true
    data.text_box.style.horizontally_stretchable = true
    data.text_box.style.vertically_stretchable = true
    data.text_box.style.size = { width = 600, height = 400 }
    data.text_box.style.rich_text_setting = defines.rich_text_setting.disabled
    data.title_log_btn.sprite = "dps_log_white"

    rebuild_log_text(data)

end



local function text_box_create_if_nessesary(player)
    local data = get_player_data(player)

    if not data.text_box and data.log_visible then
        text_box_create(data)
    end
end


local function text_box_toggle(player)
    local data = get_player_data(player)

    if valid(data.text_box) then
        data.text_box_flow.destroy()
        data.text_box_flow = nil
        data.log_visible = false
        data.title_log_btn.sprite = "dps_log_black"
    else
        text_box_create(data)

        data.log_visible = true
        
    end
    
end


local CauseType = {
    Character = 1,
    Player = 2,
    PlayerNo = 3,
    Enemy = 4,
    Any = 5
}


local TargetType = {
    Enemy = 1,
    NotAlly = 2,
    Any = 3
}




local function add_log_message(data, message)
    if not data or not valid(data.text_box) then return end
    data.messages = data.messages or {}
    local msgs = data.messages
    local start_index = data.messages_start or 1
    msgs[#msgs+1] = message
    local logical_length = #msgs - start_index + 1
    if logical_length > LOG_CAPACITY then
        -- advance start pointer (ring buffer semantics)
        data.messages_start = start_index + 1
        msgs[start_index] = nil -- allow GC, keep array mostly append-only
        -- (optional compaction when wasted head grows large)
        if data.messages_start > 128 and data.messages_start > (#msgs / 2) then
            local new = {}
            for i = data.messages_start, #msgs do new[#new+1] = msgs[i] end
            data.messages = new
            data.messages_start = 1
        end
    else
        data.messages_start = start_index
    end
    rebuild_log_text(data)
end


-- Track damage dealt by player
local function dps_entity_damaged_handler(event)
    local cause = event.cause
    local target = event.entity
    local damage_orig = event.original_damage_amount
    local damage = event.final_damage_amount
    local damage_type = event.damage_type.name
    local tick = event.tick
    local sec = tick / 60

    if damage <= 0 then return end

    -- Iterate only active players (those with an open DPS window)
    if storage.active_players then
        for player_index, _ in pairs(storage.active_players) do
            local player = game.get_player(player_index)
            if not player then goto continue end
            local data = storage.players[player_index]
            if not data or not data.dps_window_open then goto continue end

        local player_force = player.force
        local player_surface = player.surface
    -- use cached indices instead of reading GUI elements each event
    local cause_mode = data.cause_selected_index or 1
    local target_mode = data.target_selected_index or 1

        local surface_ok = true
        if (not data.all_surfaces) and target and target.valid and target.surface ~= player_surface then
            surface_ok = false
        end

        local passed_cause = false
        if cause_mode == CauseType.Character then
            passed_cause = (cause and (cause.type == "character" or cause == player.vehicle))
        elseif cause_mode == CauseType.Player then
            passed_cause = (cause and cause.force == player_force)
        elseif cause_mode == CauseType.PlayerNo then
            passed_cause = (not cause) or (cause and (not cause.valid or cause.force == player_force))
        elseif cause_mode == CauseType.Enemy then
            passed_cause = (cause and cause.force and cause.force.name == "enemy")
        elseif cause_mode == CauseType.Any then
            passed_cause = true
        end

        local passed_target = false
        if target_mode == TargetType.Enemy then
            passed_target = (target and target.force and target.force.name == "enemy")
        elseif target_mode == TargetType.NotAlly then
            passed_target = (target and target.force and not player_force.get_cease_fire(target.force.index))
        elseif target_mode == TargetType.Any then
            passed_target = true
        end

        -- Logging both accepted (+) and rejected (-)
        if not data.logging_paused and valid(data.text_box) then
            local log_msg = string.format("%.3f: %s %s #%s -> %s %s #%s, %s=%.1f->%.1f",
                sec,
                cause and cause.force and cause.force.name or "nil", 
                cause and cause.name or "nil", 
                cause and cause.unit_number or "nil",
                target and target.force and target.force.name or "nil", 
                target and target.name, 
                target and target.unit_number or "nil", 
                damage_type, damage, damage_orig)
            if surface_ok and passed_cause and passed_target then
                add_log_message(data, "[+] " .. log_msg)
            else
                add_log_message(data, "[-] " .. log_msg)
            end
        end

        if not (surface_ok and passed_cause and passed_target) then
            goto continue
        end

        local by_type = data.damage_by_type
        if not by_type then
            by_type = {}
            data.damage_by_type = by_type
        end
        local type_entry = by_type[damage_type]
        if not type_entry then
            type_entry = { pending_damage = damage, window_sums = {}, window_sums_start = 1, window_sums_tail = 0, window_total = 0, last_seen = tick }
            by_type[damage_type] = type_entry
        else
            type_entry.pending_damage = (type_entry.pending_damage or 0) + damage
            type_entry.last_seen = tick
        end
            ::continue::
        end
    end
end


-- Helper: check if any player has DPS window open
local function any_dps_window_open()
    return storage.active_players and next(storage.active_players) ~= nil
end

-- Helper: update nth-tick and damage handler registration
local function dps_update_handlers()
    if any_dps_window_open() then
        if not storage.dps_nth_tick_registered or storage.dps_nth_tick_registered ~= storage.update_interval_ticks then
            -- game.print("Registering on_nth_tick handler with interval " .. storage.update_interval_ticks)
            script.on_nth_tick(nil)
            script.on_nth_tick(storage.update_interval_ticks, on_nth_tick_handler)
            storage.dps_nth_tick_registered = storage.update_interval_ticks
        end
        if not storage.dps_damage_handler_registered then
            -- game.print("Registering on_entity_damaged handler")
            script.on_event(defines.events.on_entity_damaged, dps_entity_damaged_handler)
            storage.dps_damage_handler_registered = true
        end
    else
        if storage.dps_nth_tick_registered then
            -- game.print("Unregistering on_nth_tick handler")
            script.on_nth_tick(storage.dps_nth_tick_registered, nil)
            storage.dps_nth_tick_registered = nil
        end
        if storage.dps_damage_handler_registered then
            -- game.print("Unregistering on_entity_damaged handler")
            script.on_event(defines.events.on_entity_damaged, nil)
            storage.dps_damage_handler_registered = false
        end
    end
end

script.on_load(function()

    -- Do NOT write to storage in on_load!
    if storage.dps_nth_tick_registered then
        log("[UtilityDPS] Registering on_nth_tick handler with interval " .. storage.dps_nth_tick_registered)
        script.on_nth_tick(storage.dps_nth_tick_registered, on_nth_tick_handler)
    end

    if storage.dps_damage_handler_registered then
        log("[UtilityDPS] Registering on_entity_damaged handler")
        script.on_event(defines.events.on_entity_damaged, dps_entity_damaged_handler)
    end
end)

local function create_notepad(player)
	local data = get_player_data(player)

	data.frame = player.gui.screen.add({type="frame", name="frame", direction="vertical"})
	data.frame.style.use_header_filler = true

	local title_bar = data.frame.add({type="flow", name="title_bar", direction="horizontal"})
	title_bar.drag_target = data.frame

	local title_label = title_bar.add({type="label", caption="DPS", style="frame_title", ignored_by_interaction=true})

	local title_spacer = title_bar.add({type="empty-widget", style="draggable_space_header", ignored_by_interaction=true})
	title_spacer.style.height = 24
	title_spacer.style.horizontally_stretchable = true

    data.title_settings_btn = title_bar.add{type="sprite-button", name="dps_title_settings_btn", sprite="dps_settings_black", style="frame_action_button", tooltip="Toggle Settings"}
    data.title_log_btn = title_bar.add{type="sprite-button", name="dps_title_log_btn", sprite="dps_log_black", style="frame_action_button", tooltip="Toggle Log"}
	local title_close_btn = title_bar.add({type="sprite-button", name="dps_title_close_btn", sprite="utility/close", style="frame_action_button", tooltip="Close"})

    data.horizontal_pane = data.frame.add({type="flow", name="horizontal_pane", direction="horizontal", horizontally_stretchable = true})

    local vertical_pane = data.horizontal_pane.add({type="flow", name="vertical_pane", direction="vertical", horizontally_stretchable = true})

    local vertical_pane_dps = vertical_pane.add({type="flow", name="vertical_pane_dps", direction="vertical"})
    data.dps_label = vertical_pane_dps.add({type="label"})
    data.dps_label.style.single_line = false
    data.dps_label.caption = "No damage"

    data.vertical_pane_settings = vertical_pane.add({type="flow", name="vertical_pane_settings", direction="vertical", horizontally_stretchable = true})
    data.vertical_pane_settings.visible = data.settings_visible
    if data.settings_visible then
        data.title_settings_btn.sprite = "dps_settings_white"
    else
        data.title_settings_btn.sprite = "dps_settings_black"
    end

    local label = data.vertical_pane_settings.add({type="label", caption="Settings", style="frame_title"})
    --local switch = data.vertical_pane_settings.add({type="switch", name="dps_switch", allow_none_state=true,
    --    switch_state="left", left_label_caption="Show All", right_label_caption="Show By Type"})


    local filters_table = data.vertical_pane_settings.add({type="table", name="dps_table", column_count=2, vertical_centering=false, horizontally_stretchable = true})
    filters_table.style.horizontally_stretchable = true
    --local column_width = 120
    local cause_label = filters_table.add({type="label", caption="Cause:", horizontally_stretchable = true, horizontally_squashable = true})
    --cause_label.style.width = column_width
    local target_label = filters_table.add({type="label", caption="Target:", horizontally_stretchable = true, horizontally_squashable = true})
    --target_label.style.width = column_width

    data.cause_chosen = filters_table.add({type="list-box", name="dps_cause_chosen", allow_none_state=false, horizontally_stretchable = true, horizontally_squashable = true})
    data.cause_chosen.items = CAUSE_ITEMS
    data.cause_chosen.selected_index = data.cause_selected_index or 1
    --data.cause_chosen.style.width = column_width

    data.target_chosen = filters_table.add({type="list-box", name="dps_target_chosen", allow_none_state=false, horizontally_stretchable = true, horizontally_squashable = true})
    data.target_chosen.items = TARGET_ITEMS
    data.target_chosen.selected_index = data.target_selected_index or 1
    --data.target_chosen.style.width = column_width

    local all_surfaces_checkbox = data.vertical_pane_settings.add({type="checkbox", name="dps_all_surfaces_checkbox", caption="All Surfaces", state = data.all_surfaces or false})
    local line = data.vertical_pane_settings.add({type="line", direction="horizontal"})


    local table_dropdowns = data.vertical_pane_settings.add({type="table", column_count=2, vertical_centering=true})

    local window_dropdown_title = table_dropdowns.add({type="label", caption="window (sec)"})
    local widx = window_index_for(data.dps_window_seconds)
    data.window_dropdown = table_dropdowns.add({type="drop-down", name="dps_window_dropdown", items=WINDOW_ITEMS, selected_index=widx})

    local interval_dropdown_title = table_dropdowns.add({type="label", caption="interval (ticks)"})
    local idx = interval_index_for(storage.update_interval_ticks)
    data.interval_dropdown = table_dropdowns.add({type="drop-down", name="dps_interval_dropdown", items=INTERVAL_ITEMS, selected_index=idx})

    text_box_create_if_nessesary(player)

	if data.has_location then
		data.frame.location = { data.location_x, data.location_y }
	else
		data.frame.force_auto_center()
	end
	
    data.dps_window_open = true
    -- mark active
    storage.active_players = storage.active_players or {}
    storage.active_players[player.index] = true
    player.set_shortcut_toggled("dps-gui-shortcut", data.dps_window_open)
    dps_update_handlers()
end


local function clear_damage_data(player)
    local data = get_player_data(player)
    if data then
        data.damage_by_type = {}
    end
end

-- checked state is changed (related to checkboxes and radio buttons)
script.on_event(defines.events.on_gui_checked_state_changed, function(event)
    local element = event.element
    if not valid(element) then return end

    local player, data = get_player_and_data(event.player_index)
    if not player or not data then return end
    
    if element.name == "dps_all_surfaces_checkbox" then
        data.all_surfaces = element.state
        clear_damage_data(player)
    end
end)


-- selection state is changed (related to drop-downs and listboxes)
script.on_event(defines.events.on_gui_selection_state_changed, function(event)
    local element = event.element
    if not valid(element) then return end

    local player, data = get_player_and_data(event.player_index)
    if not player or not data then return end

    if element.name == "dps_cause_chosen" then
        data.cause_selected_index = element.selected_index
        clear_damage_data(player)
    elseif element.name == "dps_target_chosen" then
        data.target_selected_index = element.selected_index
        clear_damage_data(player)
    elseif element.name == "dps_interval_dropdown" then
        local idx = element.selected_index
        if idx < 1 then idx = 1 elseif idx > #ALLOWED_INTERVAL_TICKS then idx = #ALLOWED_INTERVAL_TICKS end
        local new_interval = ALLOWED_INTERVAL_TICKS[idx]

        storage.update_interval_ticks = new_interval
        
        -- propagate to other players' dropdowns
        for _, pdata in pairs(storage.players) do
            if pdata and valid(pdata.interval_dropdown) then
                local pidx = pdata.interval_dropdown.selected_index
                if pidx ~= idx then
                    pdata.interval_dropdown.selected_index = idx
                end               
            end
        end
        dps_update_handlers()
    
    elseif element.name == "dps_window_dropdown" then
        local widx = element.selected_index
        if widx < 1 then widx = 1 elseif widx > #ALLOWED_WINDOW_SECONDS then widx = #ALLOWED_WINDOW_SECONDS end
        data.dps_window_seconds = ALLOWED_WINDOW_SECONDS[widx]
    end
end)


local function close_notepad(player)
	local data = get_player_data(player)

	if data.frame then
		if data.frame.valid then
			data.frame.destroy()
		end
		data.frame = nil
		data.text_box = nil
        data.text_box_flow = nil
	end
    data.dps_window_open = false
    -- unmark active
    if storage.active_players then
        storage.active_players[player.index] = nil
    end
    player.set_shortcut_toggled("dps-gui-shortcut", data.dps_window_open)
    dps_update_handlers()
    clear_damage_data(player)
    
end


local function toggle_notepad(player)
	local data = get_player_data(player)

	if data and data.frame then
		close_notepad(player)
		return
	end

	create_notepad(player)
end



-- Handle on_gui_click
script.on_event(defines.events.on_gui_click, function(event)
    local element = event.element
    if not valid(element) then return end

    local player, data = get_player_and_data(event.player_index)
    if not player or not data then return end


    if element.name=="dps_title_close_btn" then
		close_notepad(player)
    elseif element.name=="dps_title_log_btn" then
        text_box_toggle(player)
    elseif element.name=="dps_title_settings_btn" then
        if valid(data.vertical_pane_settings) then
            data.settings_visible = not data.settings_visible
            data.vertical_pane_settings.visible = data.settings_visible
            data.title_settings_btn.sprite = data.settings_visible and "dps_settings_white" or "dps_settings_black"
        end
    elseif element.name == "log_toggle_btn" then
        data.logging_paused = not data.logging_paused
        element.sprite = data.logging_paused and "dps_pause" or "dps_play"
        element.tooltip = data.logging_paused and "Paused Logging" or "Running Logging"
    elseif element.name == "log_clear_btn" then
        data.messages = {}
        data.messages_start = 1
        if valid(data.text_box) then
            data.text_box.text = ""
        end
    end
end)


local function handle_dps_shortcut(event)
    -- Only open if shortcut or custom input is triggered
    if event.input_name == "dps-gui-toggle-input" or event.prototype_name == "dps-gui-shortcut" then
        local player = game.get_player(event.player_index)
        if not player then return end
        toggle_notepad(player)
    end
end

script.on_event({defines.events.on_lua_shortcut, "dps-gui-toggle-input"}, handle_dps_shortcut)