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

const SATOSHI_PER_MIN = 5.7

$(document).ready(async () => {
    const creds = nacl.box.keyPair()
    $("#pubkey").text(toHexString(creds.publicKey))
    const config = await generateConfigZip(creds)
    const link = $("#downloadConfig")
    link.attr("href", URL.createObjectURL(config))
    link.attr("download", `metervpn-config-${toHexString(creds.publicKey).slice(0, 32)}.zip`)
})

async function generateConfigZip({publicKey, secretKey}) {
    const {ip} = await $.getJSON(`/peer/${toHexString(publicKey)}`)
    const tunnelConf = `\
[Interface]
PrivateKey = ${toBase64(secretKey)}
Address = ${ip}/32
DNS = 1.1.1.1

[Peer]
PublicKey = 1t54yXxhTvUHqQE1Wh0nKqieksYm5o/KlpfQI5QUX2I=
AllowedIPs = 0.0.0.0/0
Endpoint = 159.89.121.214:52800
`
    const zip = new JSZip()
    zip.file("metervpn-1.conf", tunnelConf)
    return zip.generateAsync({type: "blob"})
}
