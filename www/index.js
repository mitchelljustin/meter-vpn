$(document).ready(async () => {
    $('[data-toggle="tooltip"]').tooltip()

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
