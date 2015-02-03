forwards = 68;
forwards2 = 0;
backwards = 65;
backwards2 = 0;
jump = 87;
jump2 = 32;
attack = 16;
attack2 = 0;
down = 115;
down2 = 0;
ws = new WebSocket("ws://" + location.host + "/tutsocket");
ws.onopen = function() {
};
ws.onmessage = function(evt) {
    msg = JSON.parse(evt.data);
    //console.log(msg);
    //console.log(msg.Sprites)
    if (msg.Info != undefined && msg.Info.length > 0) {
        if (msg.Info === "gg") {
            $('#gg').modal()
        }
    }
    var h = ($(window).height()/2-32);
    var w = ($(window).width()/2-32);
    $("#" + me).css("left", w);
    $("#" + me).css("bottom", h);
    if (msg.Flag != undefined) {
        flag = $("#flag")
        if (flag.length === 0) {
            $("#lvl").append("<img id='flag' style='left:-10000px;bottom:-10000px;' src='/images/orangered.png'></img>");
        }
        flag.css("left", msg.Flag.X);
        flag.css("bottom", msg.Flag.Y);
    }
    if (msg.Movers != undefined) {
        for (var i=0;i<msg.Movers.length;i++){
            $("#"+msg.Movers[i].Id).css("left",msg.Movers[i].X);
        }
    }
    if (msg.Chars != null) {
        names = Object.keys(msg.Chars)
        for (var i=0; i < names.length;i++) {
            if (msg.Chars[names[i]].Name === me) {
                var MeX = msg.Chars[names[i]].X
                var MeY = msg.Chars[names[i]].Y
                $("#lvl").css("left", w - MeX);
                $("#lvl").css("bottom", h - MeY);
            }
        }
        for (var i=0; i < names.length;i++) {
            if (msg.Chars[names[i]].Dead) {
                $("div#" + msg.Chars[names[i]].Name).remove();
            }
            if (msg.Chars[names[i]].Facing === "Left") {
                $("#" + msg.Chars[names[i]].Name + "> img").addClass("left");
            } else {
                $("#" + msg.Chars[names[i]].Name + "> img").removeClass("left");
            }
            if ($("#" + msg.Chars[names[i]].Name).length === 0 && !msg.Chars[names[i]].Dead) {
                if (msg.Chars[names[i]].Name != me) {
                    $("#lvl").append('<div id="'+ msg.Chars[names[i]].Name +
                        '" class="sprite ' + msg.Chars[names[i]].Type +
                        '" style="left:'+ msg.Chars[names[i]].X +
                        'px;bottom:'+ msg.Chars[names[i]].Y +
                        'px;"><h3 class="name">'+ msg.Chars[names[i]].Name +
                        '</h3><img style="position:relative;height:64px;width:64px;"src="/images/'+ msg.Chars[names[i]].Team +'Snoo.png"></img></div>');
                } else {
                    $("#lvl").append('<div id="'+ msg.Chars[names[i]].Name +
                        '" class="me sprite ' + msg.Chars[names[i]].Type +
                        '" style="left:'+ w +
                        'px;bottom:'+ h +
                        'px;"><h3 class="name">'+ msg.Chars[names[i]].Name +
                        '</h3><img style="position:relative;height:64px;width:64px;"src="/images/'+ msg.Chars[names[i]].Team +'Snoo.png"></img></div>');
                }
            }
            if (msg.Chars[names[i]].Name != me) {
                $("#" + msg.Chars[names[i]].Name).css("left", msg.Chars[names[i]].X);
                $("#" + msg.Chars[names[i]].Name).css("bottom", msg.Chars[names[i]].Y);
            } 
        }
    }
};
ws.onclose = function() {console.log("socket closed");};
$(document).keydown(function (event) {
    event.preventDefault();
    if (event.which == jump || event.which == jump2) {
        var key = 'jump';
        ws.send(JSON.stringify({
            key: key,
            direction: "down",
        }));
    }
    if (event.which == backwards || event.which == backwards2) {
        var key = 'backwards';
        ws.send(JSON.stringify({
            key: key,
            direction: "down",
        }));
    }
    if (event.which == down || event.which == down2) {
        var key = 'down';
        ws.send(JSON.stringify({
            key: key,
            direction: "down",
        }));
    }
    if (event.which == forwards || event.which == forwards2) {
        var key = 'forwards';
        ws.send(JSON.stringify({
            key: key,
            direction: "down",
        }));
    }
    if (event.which == attack || event.which == attack2) {
        event.preventDefault();
        var key = 'attack';
        ws.send(JSON.stringify({
            key: key,
            direction: "down",
        }));
    }
    //console.log(event.which, "pressed");
});
$(document).keyup(function (event) {
    event.preventDefault();
    if (event.which == jump || event.which == jump2) {
        var key = 'jump'
        ws.send(JSON.stringify({
            key: key,
            direction: "up"
        }));
    }
    if (event.which == backwards || event.which == backwards2) {
        var key = 'backwards'
        ws.send(JSON.stringify({
            key: key,
            direction: "up"
        }));
    }
    if (event.which == down || event.which == down2) {
        var key = 'down'
        ws.send(JSON.stringify({
            key: key,
            direction: "up"
        }));
    }
    if (event.which == forwards || event.which == forwards2) {
        var key = 'forwards'
        ws.send(JSON.stringify({
            key: key,
            direction: "up"
        }));
    }
    if (event.which == attack || event.which == attack2) {
        var key = 'attack';
        ws.send(JSON.stringify({
            key: key,
            direction: "up",
        }));
    }
    //console.log(event.which);
});
