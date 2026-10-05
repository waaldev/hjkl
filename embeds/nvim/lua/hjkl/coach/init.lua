local M = {}

local default_events = vim.fn.expand("~/.local/share/hjkl/coach-events.jsonl")

local cfg = {
  events_path = default_events,
  quiet = false,
  hint_throttle_ms = 8000,
}

local last_hint = 0
local stream = {}

local function now_ms()
  if vim.uv and vim.uv.now then
    return vim.uv.now()
  end
  return vim.loop.now()
end

local function append_event(ev)
  ev.ts = os.date("!%Y-%m-%dT%H:%M:%SZ")
  ev.filetype = vim.bo.filetype or ""
  if not ev.count or ev.count == 0 then
    ev.count = 1
  end
  local dir = vim.fn.fnamemodify(cfg.events_path, ":h")
  vim.fn.mkdir(dir, "p")
  local f = io.open(cfg.events_path, "a")
  if not f then
    return
  end
  f:write(vim.json.encode(ev) .. "\n")
  f:close()
end

local function hint(msg)
  if cfg.quiet then
    return
  end
  local t = now_ms()
  if t - last_hint < (cfg.hint_throttle_ms or 8000) then
    return
  end
  last_hint = t
  vim.notify("hjkl: " .. msg, vim.log.levels.INFO, { title = "hjkl coach" })
end

local function join(toks)
  return table.concat(toks, "")
end

local function emit(pattern, skill, msg, count, keys)
  append_event({
    pattern = pattern,
    skill = skill,
    keys = keys,
    count = count,
  })
  hint(msg)
end

local function detect(toks)
  local s = join(toks)
  local function run(key, n, pattern, skill, msg)
    local best, cur = 0, 0
    for _, t in ipairs(toks) do
      if t == key then
        cur = cur + 1
        if cur > best then
          best = cur
        end
      else
        cur = 0
      end
    end
    if best >= n then
      emit(pattern, skill, msg, best, string.rep(key, best))
      return true
    end
    return false
  end

  if run("j", 5, "repeat-j", "hjkl", "try a count (5j) or /search instead of jjjjj") then
    return
  end
  if run("k", 5, "repeat-k", "hjkl", "try a count (5k) or ?search instead of kkkkk") then
    return
  end
  if run("h", 5, "repeat-h", "line-ends", "try 0 ^ b instead of hhhhh") then
    return
  end
  if run("l", 5, "repeat-l", "line-ends", "try $ e w or f{char} instead of lllll") then
    return
  end
  if run("x", 4, "repeat-x", "operators", "try dw / diw instead of xxxx") then
    return
  end
  if run("w", 5, "repeat-w", "search", "try / or f{char} instead of wwwww") then
    return
  end

  local arrows = 0
  for _, t in ipairs(toks) do
    if t == "<Left>" or t == "<Right>" or t == "<Up>" or t == "<Down>" then
      arrows = arrows + 1
    end
  end
  if arrows >= 3 then
    emit("arrow-keys", "hjkl", "hjkl beats arrow keys - hands stay on the home row", arrows, "arrows")
    return
  end

  if s:find("viwd", 1, true) then
    emit("visual-iw-delete", "text-objects", "diw repeats with . - viwd does not", 1, "viwd")
    return
  end
  if s:find("viwy", 1, true) then
    emit("visual-iw-yank", "text-objects", "yiw is the operator form", 1, "viwy")
    return
  end
  if s:find("Vjjj", 1, true) then
    emit("visual-line-walk", "operators", "2dd or dj beats Vjjjd", 1, "Vjjj")
    return
  end
end

local function keytrans(typed)
  local ok, s = pcall(vim.fn.keytrans, typed)
  if ok and s and s ~= "" then
    return s
  end
  return typed
end

function M.setup(opts)
  cfg = vim.tbl_deep_extend("force", cfg, opts or {})
  vim.on_key(function(_, typed)
    if typed == nil or typed == "" then
      return
    end
    local mode = vim.fn.mode()
    local trans = keytrans(typed)
    if mode == "i" and (trans == "<Left>" or trans == "<Right>" or trans == "<Up>" or trans == "<Down>") then
      emit("insert-arrows", "modes", "Esc to Normal, move, then i/a. <C-o> for one Normal command", 1, "i+arrow")
      return
    end
    if mode ~= "n" then
      return
    end
    table.insert(stream, trans)
    if #stream > 32 then
      table.remove(stream, 1)
    end
    detect(stream)
  end)
end

return M
