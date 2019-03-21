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
