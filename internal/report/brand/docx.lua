local active = {
  prefix = os.getenv("LENS_STYLE_PREFIX"),
  logo_mm = os.getenv("LENS_LOGO_WIDTH"),
  logo_alt = os.getenv("LENS_LOGO_ALT"),
  font = os.getenv("LENS_BODY_FONT"),
  header_color = os.getenv("LENS_TABLE_INK"),
}

local function stringify(value, fallback)
  if value == nil then
    return fallback or ""
  end
  local text = pandoc.utils.stringify(value)
  if text == "" then
    return fallback or ""
  end
  return text
end

local function styled_para(text, style)
  return pandoc.Div({ pandoc.Para({ pandoc.Str(text) }) }, { ["custom-style"] = style })
end

local function cover_logo()
  return pandoc.Div({
    pandoc.Para({
      pandoc.Image({ pandoc.Str(active.logo_alt) }, "logo.png", "", { width = active.logo_mm })
    })
  }, { ["custom-style"] = active.prefix .. "CoverBrand" })
end

local function page_break()
  return pandoc.RawBlock("openxml", '<w:p><w:r><w:br w:type="page"/></w:r></w:p>')
end

local function toc_field()
  return pandoc.RawBlock("openxml", [[
<w:p>
  <w:pPr><w:pStyle w:val="]] .. active.prefix .. [[TOCTitle"/></w:pPr>
  <w:r><w:t>Contents</w:t></w:r>
</w:p>
<w:p>
  <w:r><w:fldChar w:fldCharType="begin" w:dirty="true"/></w:r>
  <w:r><w:instrText xml:space="preserve">TOC \o "1-2" \h \z \u</w:instrText></w:r>
  <w:r><w:fldChar w:fldCharType="separate"/></w:r>
  <w:r><w:t>Right-click and update field to refresh the table of contents.</w:t></w:r>
  <w:r><w:fldChar w:fldCharType="end"/></w:r>
</w:p>
]])
end


local function xml_escape(value)
  return tostring(value)
    :gsub("&", "&amp;")
    :gsub("<", "&lt;")
    :gsub(">", "&gt;")
    :gsub('"', "&quot;")
end

local function text_run(text, opts)
  opts = opts or {}
  local escaped = xml_escape(text)
  local space = ""
  if escaped:match("^%s") or escaped:match("%s$") then
    space = ' xml:space="preserve"'
  end

  local rpr = ""
  if opts.color or opts.bold then
    rpr = '<w:rPr><w:rFonts w:ascii="' .. xml_escape(active.font) .. '" w:hAnsi="' .. xml_escape(active.font) .. '" w:cs="' .. xml_escape(active.font) .. '" />'
    if opts.bold then
      rpr = rpr .. '<w:b /><w:bCs />'
    end
    if opts.color then
      rpr = rpr .. '<w:color w:val="' .. opts.color .. '" />'
    end
    rpr = rpr .. '</w:rPr>'
  end

  return "<w:r>" .. rpr .. "<w:t" .. space .. ">" .. escaped .. "</w:t></w:r>"
end

local function cell_text(cell)
  local text = pandoc.utils.stringify(cell.contents)
  text = text:gsub("\n+", " ")
  return text
end

-- Word field codes need the URL quoted; escape characters that would break the field.
local function field_url(url)
  return tostring(url):gsub('"', "%%22"):gsub("\\", "%%5C")
end

-- A clickable link inside the raw OpenXML table. Raw XML cannot add a
-- relationship for <w:hyperlink r:id>, so this uses a HYPERLINK field instead,
-- which Word and LibreOffice open as a normal hyperlink.
local function link_runs(text, target, opts)
  local instr
  if target:sub(1, 1) == "#" then
    instr = ' HYPERLINK \\l "' .. field_url(target:sub(2)) .. '" '
  else
    instr = ' HYPERLINK "' .. field_url(target) .. '" '
  end
  local run = text_run(text, opts)
  run = run:gsub("^<w:r><w:rPr>", '<w:r><w:rPr><w:rStyle w:val="Hyperlink" />', 1)
  if not run:find('w:rStyle', 1, true) then
    run = run:gsub("^<w:r>", '<w:r><w:rPr><w:rStyle w:val="Hyperlink" /></w:rPr>', 1)
  end
  return '<w:r><w:fldChar w:fldCharType="begin" /></w:r>'
    .. '<w:r><w:instrText xml:space="preserve">' .. xml_escape(instr) .. '</w:instrText></w:r>'
    .. '<w:r><w:fldChar w:fldCharType="separate" /></w:r>'
    .. run
    .. '<w:r><w:fldChar w:fldCharType="end" /></w:r>'
end

-- Same text as cell_text, but Link inlines stay clickable.
local function cell_runs(cell, opts)
  local parts = {}
  local buffer = {}

  local function flush()
    if #buffer > 0 then
      local text = table.concat(buffer):gsub("\n+", " ")
      if text ~= "" then
        table.insert(parts, text_run(text, opts))
      end
      buffer = {}
    end
  end

  local function walk_inlines(inlines)
    for _, inline in ipairs(inlines) do
      if inline.t == "Link" then
        flush()
        local text = pandoc.utils.stringify(inline.content):gsub("\n+", " ")
        if text == "" then
          text = inline.target
        end
        table.insert(parts, link_runs(text, inline.target, opts))
      elseif inline.t == "Space" or inline.t == "SoftBreak" or inline.t == "LineBreak" then
        table.insert(buffer, " ")
      elseif inline.t ~= "Note" and inline.content ~= nil and type(inline.content) ~= "string" then
        walk_inlines(inline.content)
      else
        table.insert(buffer, pandoc.utils.stringify(inline))
      end
    end
  end

  for index, block in ipairs(cell.contents) do
    if index > 1 then
      table.insert(buffer, "\n")
    end
    if (block.t == "Plain" or block.t == "Para") then
      walk_inlines(block.content)
    else
      table.insert(buffer, pandoc.utils.stringify(block))
    end
  end
  flush()

  if #parts == 0 then
    return text_run(cell_text(cell), opts)
  end
  return table.concat(parts)
end

local function table_cell(cell, is_header)
  local tcpr = '<w:tcPr><w:tcVAlign w:val="top" />'
  if cell.col_span and cell.col_span > 1 then
    tcpr = tcpr .. '<w:gridSpan w:val="' .. tostring(cell.col_span) .. '" />'
  end
  tcpr = tcpr .. '</w:tcPr>'
  local opts = {}
  if is_header then
    opts = { color = active.header_color, bold = true }
  end
  return '<w:tc>' .. tcpr .. '<w:p><w:pPr><w:pStyle w:val="Compact" /><w:keepNext /></w:pPr>' .. cell_runs(cell, opts) .. '</w:p></w:tc>'
end

local function table_row(row, is_header)
  local tr = '<w:tr>'
  if is_header then
    tr = tr .. '<w:trPr><w:tblHeader w:val="true" /></w:trPr>'
  end
  for _, cell in ipairs(row.cells) do
    tr = tr .. table_cell(cell, is_header)
  end
  return tr .. '</w:tr>'
end

local function openxml_table(tbl)
  local xml = [[<w:tbl>
<w:tblPr>
  <w:tblStyle w:val="Table" />
  <w:tblW w:w="0" w:type="auto" />
  <w:tblLayout w:type="autofit" />
  <w:tblLook w:val="04A0" w:firstRow="1" w:lastRow="0" w:firstColumn="0" w:lastColumn="0" w:noHBand="0" w:noVBand="1" />
</w:tblPr>]]

  for _, row in ipairs(tbl.head.rows) do
    xml = xml .. table_row(row, true)
  end

  for _, body in ipairs(tbl.bodies) do
    for _, row in ipairs(body.head or {}) do
      xml = xml .. table_row(row, false)
    end
    for _, row in ipairs(body.body or {}) do
      xml = xml .. table_row(row, false)
    end
  end

  for _, row in ipairs(tbl.foot.rows) do
    xml = xml .. table_row(row, false)
  end

  return xml .. '</w:tbl>'
end

-- A4 content box from the export template: 22mm side margins, 25mm top and bottom.
-- An image may use at most 70% of that width and 70% of that height.
-- A mermaid strip (diag-N.png) is already page-height. It uses the text
-- column, measured at 192 dpi because the PNG is drawn at scale 2.
local MAX_IMAGE_WIDTH_MM = 116.2
local MAX_IMAGE_HEIGHT_MM = 172.9
local DIAGRAM_WIDTH_MM = 166
local DIAGRAM_HEIGHT_MM = 190

local function read_bytes(path)
  local handle = io.open(path, "rb")
  if not handle then
    return nil
  end
  local data = handle:read("*a")
  handle:close()
  return data
end

local function be16(data, index)
  local hi, lo = data:byte(index, index + 1)
  return hi * 256 + lo
end

local function be32(data, index)
  local a, b, c, d = data:byte(index, index + 3)
  return ((a * 256 + b) * 256 + c) * 256 + d
end

local function image_pixels(path)
  local data = read_bytes(path)
  if not data then
    return nil
  end
  if data:sub(1, 8) == "\137PNG\r\n\26\n" and #data >= 24 then
    return be32(data, 17), be32(data, 21)
  end
  if data:sub(1, 2) ~= "\255\216" then
    return nil
  end
  local index = 3
  while index < #data - 8 do
    if data:byte(index) ~= 0xFF then
      return nil
    end
    local marker = data:byte(index + 1)
    if marker == 0xC0 or marker == 0xC1 or marker == 0xC2 then
      return be16(data, index + 7), be16(data, index + 5)
    end
    local length = be16(data, index + 2)
    if length < 2 then
      return nil
    end
    index = index + 2 + length
  end
  return nil
end

function Image(img)
  local width_px, height_px = image_pixels(img.src)
  local max_w, max_h, dpi = MAX_IMAGE_WIDTH_MM, MAX_IMAGE_HEIGHT_MM, 96
  if img.src:find("diag%-") then
    max_w, max_h, dpi = DIAGRAM_WIDTH_MM, DIAGRAM_HEIGHT_MM, 192
  end
  local width_mm = max_w
  if width_px and height_px and width_px > 0 and height_px > 0 then
    local natural_width = width_px / dpi * 25.4
    local natural_height = height_px / dpi * 25.4
    local scale = math.min(1, max_w / natural_width, max_h / natural_height)
    width_mm = natural_width * scale
  end
  img.attributes["width"] = string.format("%.1fmm", width_mm)
  img.attributes["height"] = nil
  return img
end

function Table(tbl)
  return pandoc.RawBlock("openxml", openxml_table(tbl))
end

function HorizontalRule()
  return {}
end

function Pandoc(doc)
  local meta = doc.meta
  local title = stringify(meta.title, "Project Export")
  local subtitle = stringify(meta.subtitle)
  local author = stringify(meta.author)
  local date = stringify(meta.date, os.date("%Y-%m-%d"))
  local client = stringify(meta.client, "")
  local project = stringify(meta.project, "")
  local licence = stringify(meta.licence)
  local disclosure = stringify(meta.disclosure)

  local blocks = {}

  table.insert(blocks, cover_logo())
  table.insert(blocks, styled_para(title, active.prefix .. "CoverTitle"))
  table.insert(blocks, styled_para(subtitle, active.prefix .. "CoverSubtitle"))
  if client ~= "" then
    table.insert(blocks, styled_para("Client: " .. client, active.prefix .. "CoverMeta"))
  end
  if project ~= "" then
    table.insert(blocks, styled_para("Project: " .. project, active.prefix .. "CoverMeta"))
  end
  table.insert(blocks, styled_para("Prepared by " .. author .. " | " .. date, active.prefix .. "CoverMeta"))
  table.insert(blocks, page_break())

  table.insert(blocks, pandoc.Header(1, "Licence and Disclosure"))
  table.insert(blocks, pandoc.Header(2, "Licence"))
  table.insert(blocks, pandoc.Para({ pandoc.Str(licence) }))
  table.insert(blocks, pandoc.Header(2, "Disclosure"))
  table.insert(blocks, pandoc.Para({ pandoc.Str(disclosure) }))
  table.insert(blocks, page_break())

  table.insert(blocks, toc_field())
  table.insert(blocks, page_break())

  for _, block in ipairs(doc.blocks) do
    if block.t ~= "HorizontalRule" then
      table.insert(blocks, block)
    end
  end

  doc.blocks = blocks
  doc.meta.title = nil
  doc.meta.subtitle = nil
  doc.meta.author = nil
  doc.meta.date = nil
  return doc
end
