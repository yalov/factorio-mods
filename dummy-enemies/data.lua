-- data.lua

local biter_name = "dummy-biter"
local worm_name = "dummy-worm"

local tint = { r = 0.95, g = 0.35, b = 0.15, a = 1.0 }
local marker = data.raw["virtual-signal"] and data.raw["virtual-signal"]["signal-D"]

-- Utility function to recursively tint graphics
local function tint_graphics(graphic, tint)
  if type(graphic) ~= "table" then return end

  if (graphic.filename or graphic.filenames or graphic.stripes) and not graphic.draw_as_shadow then
    graphic.tint = tint
  end

  for _, value in pairs(graphic) do
    tint_graphics(value, tint)
  end
end

-- Utility function to tint icons
local function tint_icon(prototype, tint)
  local icons = {}

  if prototype.icons then
    for _, layer in pairs(prototype.icons) do
      local l = table.deepcopy(layer)
      l.tint = tint
      table.insert(icons, l)
    end
  elseif prototype.icon then
    table.insert(icons, { icon = prototype.icon, icon_size = prototype.icon_size, icon_mipmaps = prototype.icon_mipmaps, tint = tint })
  end

  if marker then
    table.insert(icons, { icon = marker.icon, icon_size = marker.icon_size, icon_mipmaps = marker.icon_mipmaps, scale = 0.3125, shift = { 11, 11 } })
  end

  if #icons > 0 then
    prototype.icons = icons
    prototype.icon, prototype.icon_size, prototype.icon_mipmaps = nil, nil, nil
  end
end

-- Broad utility function to permanently mute any hidden audio maps
local function strip_all_sounds(current_table)
  if type(current_table) ~= "table" then return end
  for key, value in pairs(current_table) do
    if type(key) == "string" then
      -- Broad search
      if key:find("sound") then
        current_table[key] = nil
      end
    end
    if type(value) == "table" then
      strip_all_sounds(value)
    end
  end
end


-- 1. CLONE ENTITIES
local biter = table.deepcopy(data.raw["unit"]["small-biter"])
biter.name = biter_name
biter.order = "aa"

local worm = table.deepcopy(data.raw["turret"]["small-worm-turret"])
worm.name = worm_name
worm.order = "ab"
worm.autoplace = nil

-- 2. WORM EXCLUSIVE PROPERTIES
worm.attack_parameters.range = 0
worm.attack_parameters.cooldown = 99999999 -- Changed to force 0.00/s tooltip display
worm.attack_parameters.ammo_type.action = nil
worm.prepared_alternative_animation = nil -- Removes the random visual idle roar
worm.damaged_trigger_effect = nil

strip_all_sounds(worm)


-- 3. SHARED PROPERTIES
for _, entity in ipairs({biter, worm}) do
  entity.max_health = 10000000
  entity.healing_per_tick = 0
  entity.resistances = nil

  entity.flags = entity.flags or {}
  table.insert(entity.flags, "not-repairable")
  table.insert(entity.flags, "not-blueprintable")

  tint_graphics(entity, tint)
  tint_icon(entity, tint)
end

-- 4. REGISTER BOTH ENTITIES
data:extend{biter, worm}