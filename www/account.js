const SATOSHI_PER_HOUR = 250

function toBase64(buffer) {
    let binary = '';
    const bytes = new Uint8Array(buffer);
    const len = bytes.byteLength;
    for (let i = 0; i < len; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return window.btoa(binary);
}

async function convertToUsd(btc) {
    const data = await $.getJSON("https://api.coindesk.com/v1/bpi/currentprice/USD.json")
    const usdPrice = data.bpi.USD.rate_float
    return btc * usdPrice
}

function numberWithCommas(x) {
    return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

async function refreshDuration() {
    const {expiryDate: expiry} = await $.getJSON(`/peer`)
    const expiryDate = new Date(expiry).getTime()
    const now = new Date().getTime()
    const delta = (expiryDate - now) / 1000
    const days = Math.floor(Math.max(0, delta / 60 / 60 / 24))
    const hours = Math.floor(Math.max(0, delta / 60 / 60 % 24))
    const mins = Math.floor(Math.max(0, delta / 60 % 60))
    $("#durationDays").text(days)
    $("#durationHours").text(hours)
    $("#durationMinutes").text(mins)
}

$(document).ready(async () => {
    const payReqQrCode = new QRCode(
        document.getElementById("payReqQR"),
        {
            width: 192,
            height: 192,
        },
    )

    const $durationSelect = $("#durationSelect");

    const accountId = Cookies.get("accountId")
    $("#accountId").text(accountId)
    $("#genPayReq").click(async () => {
        const duration = String(3600 * parseInt($durationSelect.val()))
        try {
            await $.ajax({
                url: `/peer/extend`,
                type: "POST",
                dataType: "json",
                data: JSON.stringify({duration}),
            })
        } catch (e) {
            const payReq = e.responseText;
            const payReqUrl = `lightning:${payReq}`
            const $payReqStr = $("#payReqStr");
            $payReqStr.text("")
            $payReqStr.append(`<a href="${payReqUrl}">${payReq}</a>`)
            payReqQrCode.clear()
            payReqQrCode.makeCode(payReqUrl)

        }
    })

    $durationSelect.change(async () => {
        let hours = parseInt($durationSelect.val())
        if (isNaN(hours)) {
            hours = 0
        }
        const sats = Math.ceil(SATOSHI_PER_HOUR * hours)
        const btc = sats / 1e8
        const usd = await convertToUsd(btc)
        $("#btcCost").text(btc.toFixed(8))
        $("#satCost").text(numberWithCommas(sats))
        $("#usdCost").text(`$${usd.toFixed(4)}`)
    })
    await refreshDuration()
    setTimeout(refreshDuration, 1000 * 60)

    $("#genWireGuardConfig").click(async () => {
        const {publicKey, secretKey} = nacl.box.keyPair()
        await $.ajax({
            type: "POST",
            url: "/peer/pubkey",
            data: JSON.stringify({
                publicKey: toBase64(publicKey),
            }),
            dataType: "json",
        })
        const zip = new JSZip()
        const {ipv4} = await $.getJSON("/peer/ip")
        zip.file(`metervpn-toronto-ca-${accountId}.conf`, configTemplateIPv4({
            secretKey: toBase64(secretKey),
            ipv4,
        }))
        const blob = await zip.generateAsync({type: "blob"})
        saveAs(blob, "wireguard-config.zip")
    })
})

const configTemplateIPv4 = ({secretKey, ipv4}) => `\
[Interface]
PrivateKey = ${secretKey}
Address = ${ipv4}/32
DNS = 1.1.1.1

[Peer]
PublicKey = 1t54yXxhTvUHqQE1Wh0nKqieksYm5o/KlpfQI5QUX2I=
AllowedIPs = 0.0.0.0/0
Endpoint = 159.89.121.214:52800
`
