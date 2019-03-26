$(document).ready(async () => {
    $('[data-toggle="tooltip"]').tooltip()

    const prices = await $.getJSON("/price")
    $("#24hPriceSats").text(prices.satoshi.day.split(".")[0])
    $("#24hPriceUsd").text(prices.usd.day)
})
