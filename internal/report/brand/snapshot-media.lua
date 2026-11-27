-- Pandoc's sandbox disables filesystem reads by its DOCX writer. Install only
-- the media Lens generated in this private job directory before other filters.
local mime = {png="image/png", jpg="image/jpeg", jpeg="image/jpeg", gif="image/gif"}
for _, name in ipairs(pandoc.system.list_directory(".")) do
  local ext = name:match("^img%-%d+%.([a-z]+)$")
  if name == "logo.png" or name:match("^diag%-%d+%.png$") or name:match("^diag%-%d+%-%d+%.png$") then
    ext = "png"
  end
  if mime[ext] then
    local file = assert(io.open(name, "rb"))
    pandoc.mediabag.insert(name, mime[ext], file:read("*a"))
    file:close()
  end
end
