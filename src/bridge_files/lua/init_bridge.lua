local session_id, channel_id, socket_path, browser_script_path, should_map_ports, nvrh_server_info =
  ...

local should_initialize = _G._nvrh == nil

---vim.print("Preparing remote nvim", {
---	session_id = session_id,
---	channel_id = channel_id,
---	socket_path = socket_path,
---	browser_script_path = browser_script_path,
---	should_map_ports = should_map_ports,
---	should_initialize = should_initialize,
---})
