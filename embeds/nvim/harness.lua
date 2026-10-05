-- hjkl harness: one challenge in a real Neovim.
-- Env:
--   HJKL_CHALLENGE_PATH  JSON
--   HJKL_RESULT_PATH     result JSON
--   HJKL_MODE            play | verify

vim.opt.compatible = false
vim.opt.swapfile = false
vim.opt.backup = false
vim.opt.writebackup = false
vim.opt.undofile = false
vim.opt.hidden = true
vim.opt.number = true
vim.opt.relativenumber = false
vim.opt.wrap = false
vim.opt.showmode = true
vim.opt.showcmd = true
vim.opt.laststatus = 3
vim.opt.cmdheight = 1
vim.opt.shortmess:append("I")
vim.opt.backspace = "indent,eol,start"
vim.opt.mouse = ""
vim.opt.timeout = true
vim.opt.ttimeout = true
vim.opt.timeoutlen = 400
vim.opt.ttimeoutlen = 50
vim.opt.hlsearch = true
vim.opt.incsearch = true
vim.opt.virtualedit = ""
vim.g.mapleader = "\\"

local function die(msg)
  io.stderr:write("hjkl harness: " .. msg .. "\n")
  vim.cmd("cquit 1")
end

local challenge_path = vim.env.HJKL_CHALLENGE_PATH
local result_path = vim.env.HJKL_RESULT_PATH
local mode = vim.env.HJKL_MODE or "play"
if not challenge_path or not result_path then
  die("HJKL_CHALLENGE_PATH and HJKL_RESULT_PATH are required")
end

local function read_all(path)
  local f, err = io.open(path, "r")
  if not f then
    die("cannot read " .. path .. ": " .. tostring(err))
  end
  local data = f:read("*a")
  f:close()
  return data
end

local okj, challenge = pcall(vim.json.decode, read_all(challenge_path))
if not okj or type(challenge) ~= "table" then
  die("invalid challenge JSON")
end

local function split_lines(s)
  s = s or ""
  s = s:gsub("\r\n", "\n"):gsub("\r", "\n")
  if s:sub(-1) == "\n" then
    s = s:sub(1, -2)
  end
  if s == "" then
    return { "" }
  end
  return vim.split(s, "\n", { plain = true })
end

local start_lines = split_lines(challenge.start)
local target_lines = split_lines(challenge.target or challenge.start)
local start_cursor = challenge.start_cursor or { 1, 1 }
local target_cursor = challenge.target_cursor
local kind = challenge.type or "transform"

local recorded = {}
-- Keys typed outside Insert/Cmdline. Used for the Grammar Grid and technique
-- rules so text you type (e.g. "yes") never counts as a command.
local cmd_recorded = {}
local started_at = vim.uv.hrtime()
local finished = false
local ns = vim.api.nvim_create_namespace("hjkl")

local work_buf, target_buf, work_win

local function keytrans(k)
  local ok, s = pcall(vim.fn.keytrans, k)
  if ok and type(s) == "string" and s ~= "" then
    return s
  end
  return k
end

local function buf_lines(buf)
  return vim.api.nvim_buf_get_lines(buf, 0, -1, false)
end

local function lines_equal(a, b)
  if #a ~= #b then
    return false
  end
  for i = 1, #a do
    if a[i] ~= b[i] then
      return false
    end
  end
  return true
end

local function apply_cursor(win, cursor)
  if not cursor or #cursor < 2 then
    return
  end
  local buf = vim.api.nvim_win_get_buf(win)
  local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  local line = math.max(1, math.min(cursor[1], #lines))
  local maxcol = #lines[line]
  local c0 = math.max(0, math.min((cursor[2] or 1) - 1, maxcol))
  vim.api.nvim_win_set_cursor(win, { line, c0 })
end

local function cursor_matches()
  if type(target_cursor) ~= "table" or #target_cursor < 2 then
    return true
  end
  local pos = vim.api.nvim_win_get_cursor(work_win)
  return pos[1] == target_cursor[1] and (pos[2] + 1) == target_cursor[2]
end

local function buffer_matches()
  return lines_equal(buf_lines(work_buf), target_lines)
end

local function in_normal()
  return vim.api.nvim_get_mode().mode == "n"
end

-- Modes whose keys are commands rather than typed text.
local function is_cmd_mode(m)
  local c = m:sub(1, 1)
  return c == "n" or c == "v" or c == "V" or c == "\22"
end

local function won()
  -- Only Normal mode can win: a change is not finished until <Esc>.
  if not in_normal() then
    return false
  end
  if kind == "navigate" then
    return cursor_matches()
  end
  if not buffer_matches() then
    return false
  end
  if type(target_cursor) == "table" and #target_cursor >= 2 then
    return cursor_matches()
  end
  return true
end

local function duration_ms()
  return math.floor((vim.uv.hrtime() - started_at) / 1e6)
end

local function write_result(payload)
  local f, err = io.open(result_path, "w")
  if not f then
    die("cannot write result: " .. tostring(err))
  end
  f:write(vim.json.encode(payload))
  f:close()
end

local function finish(ok, reason)
  if finished then
    return
  end
  finished = true
  local pos = vim.api.nvim_win_get_cursor(work_win)
  write_result({
    ok = ok,
    keys = table.concat(recorded, ""),
    key_count = #recorded,
    cmd_keys = table.concat(cmd_recorded, ""),
    par = challenge.par or 0,
    duration_ms = duration_ms(),
    buffer = table.concat(buf_lines(work_buf), "\n"),
    cursor = { pos[1], pos[2] + 1 },
    reason = reason or "",
    aborted = not ok,
  })
  vim.schedule(function()
    if ok then
      vim.cmd("qa!")
    else
      vim.cmd("cquit 1")
    end
  end)
end

vim.keymap.set({ "n", "i", "v" }, "<F10>", function()
  finish(false, "aborted")
end, { desc = "hjkl: abort", silent = true })

vim.keymap.set("n", "<F1>", function()
  vim.notify("hint: " .. (challenge.hint or "No hint."), vim.log.levels.INFO)
end, { desc = "hjkl: hint", silent = true })

vim.keymap.set("n", "ZZ", "<Nop>")
vim.keymap.set("n", "ZQ", "<Nop>")

vim.api.nvim_create_user_command("HJKLAbort", function()
  finish(false, "aborted")
end, {})

local function setup_buffers()
  -- Drop the empty unnamed buffer nvim started with after we are ready.
  work_buf = vim.api.nvim_create_buf(true, false)
  vim.api.nvim_buf_set_name(work_buf, "hjkl-work")
  vim.api.nvim_buf_set_lines(work_buf, 0, -1, false, start_lines)
  vim.bo[work_buf].swapfile = false
  vim.bo[work_buf].modifiable = true
  vim.bo[work_buf].filetype = challenge.language or ""
  vim.bo[work_buf].undolevels = 1000

  vim.api.nvim_win_set_buf(0, work_buf)
  work_win = vim.api.nvim_get_current_win()
  apply_cursor(work_win, start_cursor)

  if kind ~= "navigate" then
    target_buf = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(target_buf, "hjkl-target")
    vim.api.nvim_buf_set_lines(target_buf, 0, -1, false, target_lines)
    vim.bo[target_buf].modifiable = false
    vim.bo[target_buf].readonly = true
    vim.bo[target_buf].bufhidden = "wipe"
    vim.bo[target_buf].filetype = challenge.language or ""

    vim.cmd("vsplit")
    local twin = vim.api.nvim_get_current_win()
    vim.api.nvim_win_set_buf(twin, target_buf)
    vim.wo[twin].winbar = " TARGET (read-only) "
    vim.wo[twin].number = true
    pcall(vim.cmd, "diffthis")
    vim.api.nvim_set_current_win(work_win)
    pcall(vim.cmd, "diffthis")
  elseif type(target_cursor) == "table" then
    local l, c = target_cursor[1], target_cursor[2]
    pcall(vim.api.nvim_buf_add_highlight, work_buf, ns, "Search", l - 1, math.max(0, c - 1), c)
    vim.fn.sign_define("HjklTarget", { text = "▶", texthl = "Search" })
    vim.fn.sign_place(1, "hjkl", "HjklTarget", work_buf, { lnum = l })
  end

  vim.wo[work_win].winbar = string.format(
    " hjkl │ %s │ %s │ par %d │ F1 hint  F10 abort ",
    string.upper(tostring(challenge.belt or "")),
    tostring(challenge.title or ""),
    challenge.par or 0
  )

  if challenge.brief and challenge.brief ~= "" then
    vim.api.nvim_echo({ { challenge.brief, "Question" } }, false, {})
  end
end

setup_buffers()

vim.on_key(function(_, typed)
  if finished or typed == nil or typed == "" then
    return
  end
  local trans = keytrans(typed)
  if trans == "<F1>" or trans == "<F10>" then
    return
  end
  table.insert(recorded, trans)
  if is_cmd_mode(vim.api.nvim_get_mode().mode) then
    table.insert(cmd_recorded, trans)
  end
  if mode == "play" and work_win and vim.api.nvim_win_is_valid(work_win) then
    vim.wo[work_win].winbar = string.format(
      " hjkl │ %s │ %s │ keys %d / par %d │ F1 hint  F10 abort ",
      string.upper(tostring(challenge.belt or "")),
      tostring(challenge.title or ""),
      #recorded,
      challenge.par or 0
    )
  end
end, ns)

local function check_win()
  if finished then
    return
  end
  if won() then
    finish(true, "solved")
  end
end

vim.api.nvim_create_autocmd({
  "TextChanged",
  "CursorMoved",
  "InsertLeave",
  "ModeChanged",
}, {
  group = vim.api.nvim_create_augroup("hjkl_check", { clear = true }),
  callback = function()
    vim.schedule(check_win)
  end,
})

if mode == "verify" then
  vim.schedule(function()
    local solution = challenge.solution or ""
    local encoded = vim.api.nvim_replace_termcodes(solution, true, false, true)
    vim.api.nvim_feedkeys(encoded, "tx", false)
    vim.wait(1500, function()
      return finished
    end, 10)
    if not finished then
      if won() then
        finish(true, "solved")
      else
        finish(false, "solution did not reach target")
      end
    end
  end)
end
