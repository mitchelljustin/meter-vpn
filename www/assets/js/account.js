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
    $('[data-toggle="tooltip"]').tooltip()
    const accountId = Cookies.get("accountId")
    if (!accountId) {
        window.location.href = "/login"
        return
    }
    const payReqQrEl = document.getElementById("payReqQR");
    const payReqQrCode = new QRCode(
        payReqQrEl,
        {
            width: 192,
            height: 192,
        },
    )

    const $durationSelect = $("#durationSelect");

    $("#accountId").text(accountId)
    $("#genPayReq").click(async () => {
        const selectedDuration = $durationSelect.val()
        if (selectedDuration === "null") {
            return
        }
        $("#requesting").show()
        $("#notRequesting").hide()
        const duration = String(3600 * parseInt(selectedDuration))
        try {
            await $.ajax({
                url: `/peer/extend`,
                type: "POST",
                dataType: "json",
                data: JSON.stringify({duration}),
            })
        } catch (e) {
            const payReq = e.responseText
            const payReqUrl = `lightning:${payReq}`

            $("#payReqStr").val(payReq)
            const clip = new ClipboardJS("#copyPayReq")
            console.log(clip)
            clip.on("success", (e) => console.log("success", e))
            clip.on("error", console.error)

            payReqQrCode.clear()
            payReqQrCode.makeCode(payReqUrl)
            $("#payReqLink").attr("href", payReqUrl)
            $("#payReqModal").modal()
        }
        $("#requesting").hide()
        $("#notRequesting").show()
    })

    const $btcCost = $("#btcCost")
    const $satCost = $("#satCost")
    const $usdCost = $("#usdCost")
    $durationSelect.change(async () => {
        let hours = parseInt($durationSelect.val())
        if (isNaN(hours)) {
            $btcCost.text("~")
            $satCost.text("~")
            $usdCost.text("~")
            return
        }
        const prices = await $.getJSON("/price")
        const satsPerHour = parseFloat(prices.satoshi.hour)
        const usdPerHour = parseFloat(prices.usd.hour)
        const sats = Math.ceil(satsPerHour * hours)
        const btc = sats / 1e8
        const usd = usdPerHour * hours
        $btcCost.text(btc.toFixed(8))
        $satCost.text(numberWithCommas(sats))
        $usdCost.text(`$${usd.toFixed(4)}`)
    })
    await refreshDuration()
    setInterval(refreshDuration, 1000 * 15)

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
        zip.file(`MeterVPN-toronto-ca.conf`, configTemplateIPv4({
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
