local M = {}

local default_events = vim.fn.expand("~/.local/share/hjkl/coach-events.jsonl")

local cfg = {
  events_path = default_events,
  quiet = false,
  hint_throttle_ms = 8000,
  flush_ms = 60000,
}

local last_hint = 0

local function now_ms()
  if vim.uv and vim.uv.now then
    return vim.uv.now()
  end
  return vim.loop.now()
end

local function append_events(evs)
  if #evs == 0 then
    return
  end
  local dir = vim.fn.fnamemodify(cfg.events_path, ":h")
  vim.fn.mkdir(dir, "p")
  local f = io.open(cfg.events_path, "a")
  if not f then
    return
  end
  local ts = os.date("!%Y-%m-%dT%H:%M:%SZ")
  for _, ev in ipairs(evs) do
    ev.ts = ts
    if not ev.count or ev.count == 0 then
      ev.count = 1
    end
    f:write(vim.json.encode(ev) .. "\n")
  end
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

local function emit(pattern, skill, count, keys)
  append_events({ { pattern = pattern, skill = skill, keys = keys, count = count, filetype = vim.bo.filetype or "" } })
end

-- Anti-pattern: the same key repeated. The hint shows when the run reaches
-- its threshold; one event with the final length is written when it ends.
local runs = {
  j = { n = 5, pattern = "repeat-j", skill = "hjkl", msg = "try a count (5j) or /search instead of jjjjj" },
  k = { n = 5, pattern = "repeat-k", skill = "hjkl", msg = "try a count (5k) or ?search instead of kkkkk" },
  h = { n = 5, pattern = "repeat-h", skill = "line-ends", msg = "try 0 ^ b instead of hhhhh" },
  l = { n = 5, pattern = "repeat-l", skill = "line-ends", msg = "try $ e w or f{char} instead of lllll" },
  x = { n = 4, pattern = "repeat-x", skill = "operators", msg = "try dw / diw instead of xxxx" },
  w = { n = 5, pattern = "repeat-w", skill = "search", msg = "try / or f{char} instead of wwwww" },
  ["<arrow>"] = { n = 3, pattern = "arrow-keys", skill = "hjkl", msg = "hjkl beats arrow keys - hands stay on the home row" },
}

local arrows = { ["<Left>"] = true, ["<Right>"] = true, ["<Up>"] = true, ["<Down>"] = true }

-- Anti-pattern: a short sequence. Matched against the tail of `recent`.
local sequences = {
  { keys = "viwd", pattern = "visual-iw-delete", skill = "text-objects", msg = "diw repeats with . - viwd does not" },
  { keys = "viwy", pattern = "visual-iw-yank", skill = "text-objects", msg = "yiw is the operator form" },
  { keys = "Vjjj", pattern = "visual-line-walk", skill = "operators", msg = "2dd or dj beats Vjjjd, and . can replay it" },
}

-- Keys whose next key is an argument, not a command (f{char}, "{reg}, ...).
local arg_keys = { f = true, t = true, F = true, T = true, r = true, m = true, ["'"] = true, ["`"] = true, ['"'] = true, q = true, ["@"] = true }

-- Single-key commands worth counting when you use them in real work.
local used_single = { ["."] = true, [";"] = true, [","] = true, n = true, N = true, ["*"] = true, ["#"] = true, ["%"] = true, ["@"] = true }

local run_key, run_len = nil, 0
local recent = ""
local op_buf = nil -- operator + motion being typed, e.g. "ci" then "ciw"
local skip_arg = false
local used = {} -- keys -> count, flushed periodically

local function flush_run()
  local r = run_key and runs[run_key]
  if r and run_len >= r.n then
    emit(r.pattern, r.skill, run_len, run_key == "<arrow>" and "<arrow>" or string.rep(run_key, run_len))
  end
  run_key, run_len = nil, 0
end

local function feed_run(tok)
  if arrows[tok] then
    tok = "<arrow>"
  end
  if tok ~= run_key then
    flush_run()
    run_key, run_len = tok, 0
  end
  run_len = run_len + 1
  local r = runs[tok]
  if r and run_len == r.n then
    hint(r.msg)
  end
end

local function feed_sequence(tok)
  recent = (recent .. tok):sub(-12)
  for _, s in ipairs(sequences) do
    if recent:sub(-#s.keys) == s.keys then
      emit(s.pattern, s.skill, 1, s.keys)
      hint(s.msg)
      recent = ""
      return
    end
  end
end

local function count_used(keys)
  used[keys] = (used[keys] or 0) + 1
end

local function finish_op()
  if op_buf and #op_buf > 1 then
    count_used(op_buf)
  end
  op_buf = nil
end

function M.flush()
  flush_run()
  finish_op()
  local evs = {}
  for keys, n in pairs(used) do
    table.insert(evs, { pattern = "used", keys = keys, count = n })
  end
  used = {}
  append_events(evs)
end

local function keytrans(typed)
  local ok, s = pcall(vim.fn.keytrans, typed)
  if ok and s and s ~= "" then
    return s
  end
  return typed
end

-- on_key runs before the key is handled, so `mode` is the mode the key was
-- typed in: "n", "no" (operator pending), "v"/"V"/"^V" (Visual), "i", ...
function M._on_key(trans, mode)
  local m = mode:sub(1, 1)
  -- A key outside operator-pending mode means the last operator finished.
  if op_buf and mode:sub(1, 2) ~= "no" then
    finish_op()
  end
  if m == "i" and arrows[trans] then
    emit("insert-arrows", "modes", 1, "i+arrow")
    hint("Esc to Normal, move, then i/a. <C-o> for one Normal command")
    return
  end

  if mode:sub(1, 2) == "no" then
    -- Motion or text object for a pending operator.
    if op_buf and trans == "<Esc>" then
      op_buf = nil
    elseif op_buf then
      op_buf = op_buf .. trans
    end
    feed_sequence(trans)
    return
  end

  if m ~= "n" and m ~= "v" and m ~= "V" and m ~= "\22" then
    return
  end

  feed_run(trans)
  feed_sequence(trans)

  if m ~= "n" then
    return
  end
  if skip_arg then
    skip_arg = false
    return
  end
  if trans == "d" or trans == "c" or trans == "y" then
    op_buf = trans
  elseif used_single[trans] then
    count_used(trans)
  end
  if arg_keys[trans] then
    skip_arg = true
  end
end

function M.setup(opts)
  cfg = vim.tbl_deep_extend("force", cfg, opts or {})
  vim.on_key(function(_, typed)
    if typed == nil or typed == "" then
      return
    end
    M._on_key(keytrans(typed), vim.api.nvim_get_mode().mode)
  end)
  local timer = (vim.uv or vim.loop).new_timer()
  timer:start(cfg.flush_ms, cfg.flush_ms, vim.schedule_wrap(M.flush))
  vim.api.nvim_create_autocmd("VimLeavePre", {
    group = vim.api.nvim_create_augroup("hjkl_coach", { clear = true }),
    callback = M.flush,
  })
end

return M
