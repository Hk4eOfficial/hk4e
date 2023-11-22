-- 原神KCP协议
hk4e_kcp_protocol = Proto("HK4E_KCP", "Genshin Impact KCP Protocol")

-- 协议字段
sess = ProtoField.uint32("hk4e_kcp.sess", "sess", base.DEC)
conv = ProtoField.uint32("hk4e_kcp.conv", "conv", base.DEC)
cmd = ProtoField.uint8("hk4e_kcp.cmd", "cmd", base.DEC)
frg = ProtoField.uint8("hk4e_kcp.frg", "frg", base.DEC)
wnd = ProtoField.uint16("hk4e_kcp.wnd", "wnd", base.DEC)
ts = ProtoField.uint32("hk4e_kcp.ts", "ts", base.DEC)
sn = ProtoField.uint32("hk4e_kcp.sn", "sn", base.DEC)
una = ProtoField.uint32("hk4e_kcp.una", "una", base.DEC)
len = ProtoField.uint32("hk4e_kcp.len", "len", base.DEC)
data = ProtoField.bytes("hk4e_kcp.data", "data", base.NONE)

hk4e_kcp_protocol.fields = { sess, conv, cmd, frg, wnd, ts, sn, una, len, data }

-- 解析器
function hk4e_kcp_protocol.dissector(buffer, pinfo, tree)
    length = buffer:len()
    if length == 0 then
        return
    end

    pinfo.cols.protocol = hk4e_kcp_protocol.name

    if length == 20 then
        local enet_sess = buffer(4, 4):uint()
        local enet_conv = buffer(8, 4):uint()
        local enet_type = buffer(12, 4):uint()
        pinfo.cols.info:append(string.format(" [ENET] Sess=%u Conv=%u Type=%u", enet_sess, enet_conv, enet_type))
        return
    end

    local first_cmd_name = get_cmd_name(buffer(8, 1):le_uint())
    local first_sn = buffer(16, 4):le_uint()
    local first_una = buffer(20, 4):le_uint()
    local first_len = buffer(24, 4):le_uint()
    pinfo.cols.info:append(string.format(" [%s] Sn=%u Una=%u Len=%u", first_cmd_name, first_sn, first_una, first_len))

    -- 解析多个KCP包
    local offset = 0
    while offset < buffer:len() do
        local sess_buf = buffer(offset + 0, 4)
        local conv_buf = buffer(offset + 4, 4)
        local cmd_buf = buffer(offset + 8, 1)
        local wnd_buf = buffer(offset + 10, 2)
        local sn_buf = buffer(offset + 16, 4)
        local len_buf = buffer(offset + 24, 4)

        local cmd_name = get_cmd_name(cmd_buf:le_uint())

        local tree_title = string.format("Genshin Impact KCP Protocol")
        local subtree = tree:add(hk4e_kcp_protocol, buffer(), tree_title)
        subtree:add_le(sess, sess_buf)
        subtree:add_le(conv, conv_buf)
        subtree:add_le(cmd, cmd_buf):append_text(" (" .. cmd_name .. ")")
        subtree:add_le(frg, buffer(offset + 9, 1))
        subtree:add_le(wnd, wnd_buf)
        subtree:add_le(ts, buffer(offset + 12, 4))
        subtree:add_le(sn, sn_buf)
        subtree:add_le(una, buffer(offset + 20, 4))
        subtree:add_le(len, len_buf)

        local data_len = len_buf:le_uint()
        if data_len ~= 0 then
            local data_buf = buffer(offset + 28, data_len)
            subtree:add(data, data_buf)
        end

        offset = offset + 28 + data_len
    end
end

function get_cmd_name(cmd_val)
    if cmd_val == 81 then
        return "PSH"
    elseif cmd_val == 82 then
        return "ACK"
    elseif cmd_val == 83 then
        return "ASK"
    elseif cmd_val == 84 then
        return "TELL"
    end
end

-- 注册到UDP
local udp_port = DissectorTable.get("udp.port")
-- 注册到22222端口
udp_port:add(22222, hk4e_kcp_protocol)
