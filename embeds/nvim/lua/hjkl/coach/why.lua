-- :HjklWhy - "was there a faster way?" for edits in your own files.
--
-- The coach keeps the last edit: a snapshot of the buffer before it, the
-- buffer after it, and the keys pressed in between. An edit ends when you
-- have been idle in Normal mode for edit_idle_ms. Nothing leaves your
-- machine until you run :HjklWhy, and then only the changed lines plus a
-- little context go to `hjkl suggest`, which asks the AI and replays the
-- answer in a headless nvim before showing it.
local W = {}

local uv = vim.uv or vim.loop

local cfg = {
  hjkl_cmd = "hjkl",
  edit_idle_ms = 2000,
  context_lines = 2,
  max_lines = 40,
  offer_min_keys = 12,
  -- Files that never leave your machine, matched against the full path.
  exclude = { "*.env", "*/.env*", "*.pem", "*.key", "*secret*", "*credential*", "*password*" },
}

local hint = function() end -- set by setup: the coach's fading hint
local enabled = false -- from `hjkl suggest --status`

local snap = nil -- { buf, lines, tick, cursor }
local keys = {}
local last_edit = nil
local idle = nil

local function track_buf(buf)
  return vim.bo[buf].buftype == "" and vim.bo[buf].modifiable and vim.api.nvim_buf_get_name(buf) ~= ""
end

local function excluded(name)
  for _, glob in ipairs(cfg.exclude) do
    if vim.fn.match(name, vim.fn.glob2regpat(glob)) >= 0 then
      return true
    end
  end
  return false
end

local function take_snap()
  local buf = vim.api.nvim_get_current_buf()
  local pos = vim.api.nvim_win_get_cursor(0)
  snap = {
    buf = buf,
    lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false),
    tick = vim.b[buf].changedtick,
    cursor = { pos[1], pos[2] + 1 },
  }
  keys = {}
end

-- Saving is not part of the edit.
local function edit_keys()
  local s = table.concat(keys, "")
  for _, w in ipairs({ ":w<CR>", ":wa<CR>", ":x<CR>", ":update<CR>" }) do
    s = s:gsub(vim.pesc(w), "")
  end
  return s
end

local function key_count(notation)
  local n, i = 0, 1
  while i <= #notation do
    local angle = notation:match("^<[^<>]+>", i)
    i = i + (angle and #angle or 1)
    n = n + 1
  end
  return n
end

-- Characters actually changed, line by line (common prefix/suffix removed),
-- to judge whether an edit cost far more keys than its size.
local function changed_chars(before, after)
  local ins = 0
  for i = 1, math.max(#before, #after) do
    local a, b = before[i] or "", after[i] or ""
    if a ~= b then
      local p = 0
      while p < #a and p < #b and a:sub(p + 1, p + 1) == b:sub(p + 1, p + 1) do
        p = p + 1
      end
      local s = 0
      while s < #a - p and s < #b - p and a:sub(#a - s, #a - s) == b:sub(#b - s, #b - s) do
        s = s + 1
      end
      ins = ins + math.max(#b - p - s, 1)
    end
  end
  return ins
end

-- The changed region of a buffer, plus context and the start cursor line.
local function region(before, after, cursor_line)
  local p = 0
  while p < #before and p < #after and before[p + 1] == after[p + 1] do
    p = p + 1
  end
  local s = 0
  while s < #before - p and s < #after - p and before[#before - s] == after[#after - s] do
    s = s + 1
  end
  local lo = math.max(1, p + 1 - cfg.context_lines)
  local hi_b = math.min(#before, #before - s + cfg.context_lines)
  lo = math.min(lo, cursor_line)
  hi_b = math.max(hi_b, cursor_line)
  local tail = #before - hi_b -- identical lines below the region, in both
  local hi_a = #after - tail
  return lo, hi_b, hi_a
end

local function finish_edit()
  local buf = snap.buf
  local name = vim.api.nvim_buf_get_name(buf)
  local after_all = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  local k = edit_keys()
  local before_all = snap.lines
  local cursor = snap.cursor
  take_snap()
  if excluded(name) then
    return
  end
  local lo, hi_b, hi_a = region(before_all, after_all, cursor[1])
  if hi_b - lo + 1 > cfg.max_lines or hi_a - lo + 1 > cfg.max_lines then
    return
  end
  local before = vim.list_slice(before_all, lo, hi_b)
  local after = vim.list_slice(after_all, lo, hi_a)
  last_edit = {
    before = table.concat(before, "\n"),
    after = table.concat(after, "\n"),
    cursor = { cursor[1] - lo + 1, cursor[2] },
    keys = k,
    filetype = vim.bo[buf].filetype,
  }
  local n = key_count(k)
  if enabled and n >= cfg.offer_min_keys and n > 1.5 * changed_chars(before, after) + 6 then
    hint("costly-edit", "golf", string.format("%d keys for that edit - :HjklWhy for a shorter way", n))
  end
end

local function on_idle()
  if not snap or vim.api.nvim_get_mode().mode ~= "n" then
    return
  end
  if vim.api.nvim_get_current_buf() ~= snap.buf or not vim.api.nvim_buf_is_valid(snap.buf) then
    take_snap()
  elseif vim.b[snap.buf].changedtick ~= snap.tick then
    finish_edit()
  else
    take_snap() -- just moving around: the next edit starts here
  end
end

function W.on_key(trans)
  local buf = vim.api.nvim_get_current_buf()
  if not track_buf(buf) then
    return
  end
  if not snap or snap.buf ~= buf then
    take_snap() -- on_key runs before the key: this is the "before" state
  end
  if #keys < 400 then
    table.insert(keys, trans)
  end
  idle:stop()
  idle:start(cfg.edit_idle_ms, 0, vim.schedule_wrap(on_idle))
end

local function show(res)
  local lines = {
    string.format("  your edit: %d keys  →  %d keys", res.your_key_count or 0, res.key_count or 0),
    "",
    "    " .. res.keys,
    "",
  }
  for _, l in ipairs(vim.split(res.explanation or "", "\n")) do
    table.insert(lines, "  " .. l)
  end
  vim.list_extend(lines, {
    "",
    "  ✓ checked: replaying these keys reproduces your edit",
    "",
    "  p practice it   s save as a drill   q close",
  })
  local width = 20
  for _, l in ipairs(lines) do
    width = math.max(width, vim.fn.strdisplaywidth(l) + 2)
  end
  local buf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
  vim.bo[buf].modifiable = false
  local win = vim.api.nvim_open_win(buf, true, {
    relative = "editor",
    width = math.min(width, vim.o.columns - 4),
    height = #lines,
    row = math.floor((vim.o.lines - #lines) / 2),
    col = math.floor((vim.o.columns - width) / 2),
    style = "minimal",
    border = "rounded",
    title = " hjkl · a shorter way ",
  })
  vim.wo[win].wrap = true
  local function close()
    if vim.api.nvim_win_is_valid(win) then
      vim.api.nvim_win_close(win, true)
    end
  end
  local o = { buffer = buf, nowait = true, silent = true }
  vim.keymap.set("n", "q", close, o)
  vim.keymap.set("n", "<Esc>", close, o)
  vim.keymap.set("n", "p", function()
    close()
    vim.cmd("tabnew")
    vim.fn.termopen({ cfg.hjkl_cmd, "suggest", "--practice", res.drill }, {
      on_exit = function()
        vim.schedule(function()
          pcall(vim.cmd, "tabclose")
        end)
      end,
    })
    vim.cmd("startinsert")
  end, o)
  vim.keymap.set("n", "s", function()
    close()
    vim.system({ cfg.hjkl_cmd, "suggest", "--save", res.drill }, { text = true }, function(out)
      vim.schedule(function()
        local ok, r = pcall(vim.json.decode, out.stdout or "")
        if ok and type(r) == "table" and r.ok then
          vim.notify("hjkl: saved - it will come back in your reviews", vim.log.levels.INFO)
        else
          vim.notify("hjkl: could not save the drill", vim.log.levels.WARN)
        end
      end)
    end)
  end, o)
  W._float = { buf = buf, win = win, lines = lines }
end

function W.why()
  if not enabled then
    vim.notify(
      "hjkl: suggestions are off. Set suggest = true under [ai] in ~/.config/hjkl/config.toml.\n"
        .. "Only the changed lines of one edit are sent, and only when you run :HjklWhy.",
      vim.log.levels.INFO
    )
    return
  end
  if not last_edit then
    vim.notify("hjkl: no recent edit yet - edits are captured after a short pause in Normal mode", vim.log.levels.INFO)
    return
  end
  local path = vim.fn.tempname() .. ".json"
  vim.fn.writefile({ vim.json.encode(last_edit) }, path)
  vim.notify("hjkl: looking for a shorter way (checked in nvim before it is shown)…", vim.log.levels.INFO)
  vim.system({ cfg.hjkl_cmd, "suggest", "--input", path }, { text = true }, function(out)
    vim.schedule(function()
      os.remove(path)
      local ok, res = pcall(vim.json.decode, out.stdout or "")
      if not ok or type(res) ~= "table" then
        vim.notify("hjkl: suggest failed: " .. (out.stderr or "no output"), vim.log.levels.WARN)
      elseif not res.ok then
        vim.notify("hjkl: " .. (res.error or "no shorter way found"), vim.log.levels.INFO)
      else
        show(res)
      end
    end)
  end)
end

function W.setup(opts, hint_fn)
  cfg = vim.tbl_deep_extend("force", cfg, opts or {})
  hint = hint_fn
  idle = uv.new_timer()
  -- One switch, in hjkl's config: ask the CLI whether suggestions are on.
  pcall(vim.system, { cfg.hjkl_cmd, "suggest", "--status" }, { text = true }, function(out)
    local ok, res = pcall(vim.json.decode, out.stdout or "")
    enabled = ok and type(res) == "table" and res.enabled == true
  end)
  vim.api.nvim_create_user_command("HjklWhy", W.why, { desc = "hjkl: a verified shorter way to make your last edit" })
end

-- For tests.
function W._last_edit()
  return last_edit
end

function W._enabled()
  return enabled
end

return W
