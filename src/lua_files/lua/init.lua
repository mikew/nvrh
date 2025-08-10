_G._nvrh = {
  session_id = session_id,

  ---@type integer[]
  client_channels = {},

  ---@type { [string]: boolean }
  mapped_ports = {},
}

vim.env.NVRH_SESSION_ID = session_id
