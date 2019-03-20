function toHexString(byteArray) {
    return Array.prototype.map.call(byteArray, function (byte) {
        return ('0' + (byte & 0xFF).toString(16)).slice(-2);
    }).join('');
}

function toBase64(buffer) {
    let binary = '';
    const bytes = new Uint8Array(buffer);
    const len = bytes.byteLength;
    for (let i = 0; i < len; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return window.btoa(binary);
}

$(document).ready(async () => {
    const $accountId = $("#accountId");
    $("#createNewAccount").click(async (e) => {
        e.stopPropagation()
        if ($accountId.text().trim() !== "") {
            return
        }
        const {accountId} = await $.post("/peer")
        $("#accountIdPanel").show()
        $accountId.text(accountId)
        Cookies.set("accountId", accountId)
    })
})

const configTemplate = (secretKey, ipv4, ipv6) => `\
[Interface]
PrivateKey = ${secretKey}
Address = ${ipv4}/32, ${ipv6}/128
DNS = 1.1.1.1

[Peer]
PublicKey = 1t54yXxhTvUHqQE1Wh0nKqieksYm5o/KlpfQI5QUX2I=
AllowedIPs = 0.0.0.0/0
Endpoint = 159.89.121.214:52800
`
