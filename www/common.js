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