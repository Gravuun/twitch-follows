$(document).ready(function () { 
    $("#logout-btn").click(function () {
        $.get("/logout");
     });
});