$(document).ready(async () => {
    $("#getNewAccountId").click(async () => {
        const {accountId} = await $.post("/peer")
        Cookies.set("accountId", accountId)
        $("#withAccount").show()
        $("#accountId").text(accountId)
    })
})