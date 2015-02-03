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
stat = 9;
var offset = 0;
var attacks = 0;
var respawn = false;
var lastflag;
var flagtile;
var flag = "<div class='flag'><i class='oj fa fa-envelope-o fa-5x'></i></div>";
var rhammer = "<div class='rhammer'><i class='fa icon-arrow-right-a fa-lg'></i></div>";
var lhammer = "<div class='lhammer'><i class='fa icon-arrow-left-a fa-lg'></i></div>";

function winner(msg) {
    $("#gg").modal();
}

function clearhammers() {
    $(".rhammer").remove()
    $(".lhammer").remove()
}

function warn(txt) {
    $("#warn").html(txt);
    window.setTimeout(function () {
        clearwarn()
    }, 3000);
}

function clearwarn() {
    $("#warn").html("");
    respawn = false;
}

function subwarn(txt) {
    $("#subwarn").html(txt);
    window.setTimeout(function () {
        clearsubwarn()
    }, 3000);
}

function clearsubwarn() {
    $("#subwarn").html("");
}

var tickers = 0;
function info(txt) {
    var m = "<p id="+tickers+">"+txt+"</p>";
    tickers++;
    $("#info").append(m);
    window.setTimeout(function () {
        clearinfo(tickers-1)
    }, 3000);
}

function clearinfo(i) {
    $("p#"+i).remove();
}

function respawning() {
    respawn = true;
    warn("You've died!  Respawning soon.");
}

ws = new WebSocket("ws://" + location.host + "/tutsocket");
ws.onopen = function () {};
ws.onmessage = function (evt) {
    msg = JSON.parse(evt.data);
    //console.log(msg);
    if (graphics == "med" || graphics == "high") {
        // graphics for med or high settings - char animations, etc
    }
    if (graphics == "high") {
        // graphics for high settings - background animations, etc
    }
    if (msg.type != undefined && msg.text.length > 0) {
        //console.log(msg);
        if (msg.type === "gg") {
            winner(msg);
        } 
        if (msg.type === "warn") {
            warn(msg.text);
        }
        if (msg.type === "subwarn") {
            subwarn(msg.text);
        }
        if (msg.type === "info") {
            info(msg.text);
        }
        if (msg.type === "death") {
            $("div#" + msg.text).remove();
        }
    }
    if (msg.Msg != undefined && msg.Msg.text != undefined && msg.Msg.text.length > 0) {
        var m = msg.Msg;
        //console.log(m);
        if (m.type === "gg") {
            winner(msg);
        } 
        if (m.type === "warn") {
            warn(m.text);
        }
        if (m.type === "subwarn") {
            subwarn(m.text);
        }
        if (m.type === "info") {
            info(m.text);
        }
    }
    var h = ($(window).height() / 2 - 32);
    var w = ($(window).width() / 2 - 32);
    $("#" + me).css("left", w);
    $("#" + me).css("bottom", h);
    if (msg.Flag != undefined) {
        if (msg.Flag.Owner != "" && $(".flag").length == 0) {
                $("#" + msg.Flag.Owner).append(flag);
        }
        if (msg.Flag.Owner != "") {
            if (lastflag != msg.Flag.Owner) {
                //console.log(msg.Flag);
                flagtile = "";
                $(".flag").remove();
                $("#" + msg.Flag.Owner).append(flag);
                lastflag = msg.Flag.Owner;
            }
        }
        if (msg.Flag.Owner === "") {
            if (flagtile != msg.Flag.Tile.Id) {
                //console.log(msg.Flag);
                lastflag = "";
                $(".flag").remove();
                $("#" + msg.Flag.Tile.Id).append(flag);
                flagtile = msg.Flag.Tile.Id;
            }
        }
    }
    if (msg.Movers != undefined) {
        for (var i = 0; i < msg.Movers.length; i++) {
            $("#" + msg.Movers[i].Id).css("left", msg.Movers[i].X);
        }
    }
    if (msg.Disconnects != null) {
        names = Object.keys(msg.Disconnects)
        for (var i = 0; i < names.length; i++) {
            $("div#" + msg.Disconnects[names[i]].Name).remove();
        }
    }
    if (msg.Chars != null) {
        names = Object.keys(msg.Chars)
        for (var i = 0; i < names.length; i++) {
            if (msg.Chars[names[i]].Name === me) {
                var MeX = msg.Chars[names[i]].X
                var MeY = msg.Chars[names[i]].Y
                if (attacks != msg.Chars[names[i]].Attacks) {
                    attacks = msg.Chars[names[i]].Attacks
                    $("#attacks").html('<span>' + attacks + '</span><i class="fa fa-gavel fa-lg"></i>')
                }
                $("#lvl").css({"left": w - MeX, "bottom": h - MeY});
            }
        }
        for (var i = 0; i < names.length; i++) {
            var ch = msg.Chars[names[i]];
            if (ch.Dead) {
                $("div#" + ch.Name).remove();
                if (ch.Name == me && !respawn) {
                    respawning();
                }
            }
            if (ch.Attacked) {
                if (ch.Facing === "Left") {
                    $("#"+ch.Name).append(lhammer);
                    window.setTimeout(function () {
                        clearhammers()
                    }, 200);
                } else {
                    $("#"+ch.Name).append(rhammer);
                    window.setTimeout(function () {
                        clearhammers()
                    }, 200);
                }
            }
            if (ch.Facing === "Left") {
                if (!$("#" + ch.Name).hasClass("left")) {
                    $("#" + ch.Name).addClass("left");
                }
            } else {
                if ($("#" + ch.Name).hasClass("left")) {
                    $("#" + ch.Name).removeClass("left");
                }
            }
            if ($("#" + ch.Name).length === 0 && !ch.Dead) {
                if (ch.Name != me) {
                    $("#lvl").append('<div id="' + ch.Name +
                        '" class="Player"' +
                        'style="left:' + ch.X +
                        'px;bottom:' + ch.Y +
                        'px;"><h3 class="name">' + ch.Name +
                        '</h3><img style="position:relative;height:64px;width:64px;"src="/images/' + ch.Team + 'Snoo.png"></img></div>');
                } else {
                    $("#lvl").append('<div id="' + ch.Name +
                        '" class="me Player"' +
                        'style="left:' + w +
                        'px;bottom:' + h +
                        'px;"><h3 class="name">' + ch.Name +
                        '</h3><img style="position:relative;height:64px;width:64px;"src="/images/' + ch.Team + 'Snoo.png"></img></div>');
                }
            }
            if (ch.Name != me) {
                $("#" + ch.Name).css("left", ch.X);
                $("#" + ch.Name).css("bottom", ch.Y);
            }
        }
    }
};
ws.onclose = function () {
    console.log("socket closed");
};
$(document).keydown(function (event) {
    //event.preventDefault();
    if (event.which == stat) {
        event.preventDefault();
        $("#tab").modal({show:true});
    }
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
    //event.preventDefault();
    if (event.which == stat) {
        event.preventDefault();
        $("#tab").modal("toggle");
    }
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
