$(document).ready(async () => {
    const $accountId = $("#accountId");
    const $getNewAccountId = $("#getNewAccountId");
    $getNewAccountId.click(async () => {
        const {accountId} = await $.post("/peer")
        Cookies.set("accountId", accountId)
        $("#withAccount").removeClass("d-none")
        $accountId.text(accountId)
        $getNewAccountId.attr("disabled", true)
    })
})