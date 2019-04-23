function toBase64(buffer) {
    let binary = '';
    const bytes = new Uint8Array(buffer);
    const len = bytes.byteLength;
    for (let i = 0; i < len; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return window.btoa(binary);
}

function numberWithCommas(x) {
    return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

$(document).ready(() => {
    $(`[data-action="logout"]`).click(() => {
        gtag('event', 'click', {
            event_category: 'engagement',
            event_label: 'logout',
        })
        Cookies.remove("accountId")
        window.location.href = "/"
    })

    $('[data-action]').click(function() {
        const label = $(this).attr("data-action");
        gtag('event', 'click', {
            event_category: 'engagement',
            event_label: label,
        })
    })
})