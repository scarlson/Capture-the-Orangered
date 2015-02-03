var offset = 0;
var attacks = 0;
var respawn = false;
var flag = "<div class='flag'><i class='oj fa fa-envelope-o fa-5x'></i></div>";
var rhammer = "<div class='rhammer'><i class='fa icon-arrow-right-a fa-lg'></i></div>";
var lhammer = "<div class='lhammer'><i class='fa icon-arrow-left-a fa-lg'></i></div>";

function savesettings() {
    ws.send(JSON.stringify({
        Audio: audio,
        Music: music,
        Graphics: graphics,
        Forwards: forwards,
        Forwards2: forwards2,
        Backwards: backwards,
        Backwards2: backwards2,
        Jump: jump,
        Jump2: jump2,
        Attack: attack,
        Attack2: attack2,
        Down: down,
        Down2: down2,
        Stat: stat,
    }));
}

function winner(text) {
    msg = JSON.parse(text);
    if (msg.Orangered >= capmax) {
        $("#gg .modal-title").html("Team Orangered Wins!");
        document.title = "Orangered Wins!";
    }
    if (msg.Periwinkle >= capmax) {
        $("#gg .modal-title").html("Team Periwinkle Wins!");
        document.title = "Periwinkle Wins!";
    }
    if (msg.Orangered === undefined) {
        msg.Orangered = 0;
    }
    if (msg.Periwinkle === undefined) {
        msg.Periwinkle = 0;
    }
    $("#scoreOrangered").html("<h3>Orangered</h3><h4>" + msg.Orangered + "</h4>");
    $("#scorePeriwinkle").html("<h3>Periwinkle</h3><h4>" + msg.Periwinkle + "</h4>");
    $("#gg").modal();
}

function clearhammers(id) {
    $("#" + id + " .rhammer").remove()
    $("#" + id + " .lhammer").remove()
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
    var m = "<p>"+txt+"</p>";
    $("#info").append(m);
    window.setTimeout(function () {
        clearinfo()
    }, 3000);
}

function clearinfo() {
    $("#info").html("");
}

function respawning() {
    respawn = true;
    warn("You've died!  Respawning soon.");
}

ws = new WebSocket("ws://" + location.host + "/socket");
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
        if (msg.type === "status" && msg.text === "started") {
            document.title = "▶ " + document.title;
            warn("GO!");
            $("#Periwinkle").html("0");
            $("#Orangered").html("0");
        }
        if (msg.type === "flag") {
            if ($("#"+msg.text).children(".flag").length === 0) {
                $(".flag").remove();
                $("#" + msg.text).append(flag);
                $("#Flagger").html("");
            }
        } 
        if (msg.type === "gg") {
            winner(msg.text);
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
        if (msg.type === "delete" || msg.type === "disconnect") {
            $("#tab #" + msg.text).remove();
            $(".Player#" + msg.text).remove();
        }
        if (msg.type === "death") {
            $(".Player#" + msg.text).remove();
        }
    }
    var h = ($(window).height() / 2 - 32);
    var w = ($(window).width() / 2 - 32);
    $("#" + me).css("left", w);
    $("#" + me).css("bottom", h);
    if (msg.Orangered != lastorangered) {
        $("#Orangered").html(msg.Orangered);
        lastorangered = msg.Orangered;
    }
    if (msg.Periwinkle != lastperiwinkle) {
        $("#Periwinkle").html(msg.Periwinkle);
        lastperiwinkle = msg.Periwinkle;
    }
    if (msg.Flag != undefined) {
        //if (msg.Flag.Owner != "" && $(".flag").length == 0) {
        //        $("#" + msg.Flag.Owner).append(flag);
        //}
        if (msg.Flag.Owner != "") {
            if ($("#"+msg.Flag.Owner).children(".flag").length === 0) {
                //console.log(msg.Flag);
                $(".flag").remove();
                $("#" + msg.Flag.Owner).append(flag);
                $("#Flagger").html(msg.Flag.Owner + " has the Orangered!");
            }
            if ($("#Flagger").html().substring(0,msg.Flag.Owner.length) != msg.Flag.Owner) {
                $("#Flagger").html(msg.Flag.Owner + " has the Orangered!");
            }
        }
        if (msg.Flag.Owner === "") {
            if ($("#"+msg.Flag.Tile.Id).children(".flag").length === 0) {
                //console.log(msg.Flag);
                $(".flag").remove();
                $("#" + msg.Flag.Tile.Id).append(flag);
                $("#Flagger").html("");
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
            $("#tab" + msg.Disconnects[names[i]].Name + " #" + msg.Disconnects[names[i]].Name).remove();
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
                if ($("#Flagger").html().substring(0,ch.Name.length) === ch.Name) {
                    $("#Flagger").html("");
                }
                if (ch.Name == me && !respawn) {
                    respawning();
                }
            }
            if (ch.Attacked) {
                if (ch.Facing === "Left") {
                    $("#"+ch.Name).append(lhammer);
                    window.setTimeout(function () {
                        clearhammers(ch.Name)
                    }, 200);
                } else {
                    $("#"+ch.Name).append(rhammer);
                    window.setTimeout(function () {
                        clearhammers(ch.Name)
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
            if ($("#tab" + ch.Team + " #" + ch.Name).length === 0) { // doing the tab stuff
                $("#tab" + ch.Team).append('<p id="' + ch.Name + '">' + ch.Name + '</p>');
            }
            
            if ($(".Player#" + ch.Name).length === 0 && !ch.Dead) {
                if (ch.Name != me) {
                    $("#lvl").append('<div id="' + ch.Name +
                        '" class="Player"' +
                        'style="left:' + ch.X +
                        'px;bottom:' + ch.Y +
                        'px;"><h3 class="name">' + ch.Name +
                        '</h3><img src="/images/' + ch.Team + ch.Sprite + '.png"></img></div>');
                } else {
                    $("#lvl").append('<div id="' + ch.Name +
                        '" class="me Player"' +
                        'style="left:' + w +
                        'px;bottom:' + h +
                        'px;"><h3 class="name">' + ch.Name +
                        '</h3><img src="/images/' + ch.Team + ch.Sprite + '.png"></img></div>');
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
