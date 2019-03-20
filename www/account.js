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

function hexStringToByte(str) {
    if (!str) {
        return new Uint8Array();
    }

    var a = [];
    for (var i = 0, len = str.length; i < len; i+=2) {
        a.push(parseInt(str.substr(i,2),16));
    }

    return new Uint8Array(a);
}

async function convertToUsd(btc) {
    const data = await $.getJSON("https://api.coindesk.com/v1/bpi/currentprice/USD.json")
    const usdPrice = data.bpi.USD.rate_float
    return btc * usdPrice
}

function numberWithCommas(x) {
    return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

async function refreshDuration(accountId) {
    const {expiryDate: expiry} = await $.getJSON(`/peer/${accountId}`)
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
    const $durationSelect = $("#durationSelect");

    const accountId = Cookies.get("accountId")
    $("#accountId").text(accountId)
    $("#genPayReq").click(async () => {
        const duration = String(3600 * parseInt($durationSelect.val()))
        try {
            await $.ajax({
                url: `/peer/${accountId}/extend`,
                type: "POST",
                dataType: "json",
                data: JSON.stringify({duration}),
            })
        } catch (e) {
            const payReq = e.responseText;
            const $payReqStr = $("#payReqStr");
            $payReqStr.text("")
            $payReqStr.append(`<a href="lightning:${payReq}">${payReq}</a>`)
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
    await refreshDuration(accountId)
    setTimeout(refreshDuration, 1000 * 60)
})