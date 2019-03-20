import * as base32 from "base32"
import * as nacl from "tweetnacl"
import JSZip from "jszip"

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
    const creds = nacl.box.keyPair()
    const accountIdBytes = new Uint8Array(10)
    crypto.getRandomValues(accountIdBytes)
    const accountId = base32.encode(accountIdBytes)
    console.log(accountId)
    let pubKeyHex = toHexString(creds.publicKey);
    $("#pubkey").text(pubKeyHex)
    const config = await generateConfigZip(creds)
    const link = $("#genConfig")
    link.attr("href", URL.createObjectURL(config))
    link.attr("download", `metervpn-config-${pubKeyHex.slice(0, 32)}.zip`)
    $("#topUpKey").text(`${pubKeyHex.slice(0, 16)}...`)
    $("#goTopUp").attr("href", `/x/${pubKeyHex}`)
})

async function generateConfigZip({publicKey, secretKey}) {
    const {ipv6} = await $.getJSON(`/peer/${toHexString(publicKey)}`)
    const tunnelConf = `\
[Interface]
PrivateKey = ${toBase64(secretKey)}
Address = ${ipv6}/128
DNS = 1.1.1.1

[Peer]
PublicKey = 1t54yXxhTvUHqQE1Wh0nKqieksYm5o/KlpfQI5QUX2I=
AllowedIPs = 0.0.0.0/0
Endpoint = 159.89.121.214:52800
`
    const zip = new JSZip()
    zip.file("metervpn-toronto-ca.conf", tunnelConf)
    return zip.generateAsync({type: "blob"})
}
