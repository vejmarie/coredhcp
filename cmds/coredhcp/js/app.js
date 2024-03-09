// Copyright 2024 Hewlett Packard Development LP.

var clientList = [];

function client() {
    this.mac = null;
    this.state = null;
}


function clearDocument(){
	$(document.body).empty();
}

function loadHTML(filename){
	jQuery.ajaxSetup({async:false});
        jQuery.get(filename, function(data, status){
                $(document.body).append(data);
        });
        jQuery.ajaxSetup({async:true});
}
function addSpace(descriptor, length){
	var label ="|";
	label = label + descriptor;
	for ( let i = 0 ; i < ( length - descriptor.length); i++ ) {
		label = label + '&nbsp;';
	}
	return label
}
function updateClients() {
	var Url = '/clients';
	        $.ajax({
                type: "GET",
                contentType: 'application/json',
                url: Url,
                success: function(response){
                        var data = JSON.parse(response);
                        $("#clients").html("");
                        if ( data != null )
                        {
                                var myarray = Object.keys(data);
                                clientList = [];
                                for (let i = 0; i  < myarray.length; i++) {
                                        var newClient = new client();
                                        newClient.mac = data[myarray[i]].MacAddress;
                                        newClient.state = data[myarray[i]].State;
                                        newClient.IP= data[myarray[i]].IP;
                                        clientList.push(newClient);
                                }
                                var columnSize = [2,3,2,6];
                                var header = ["ID","Mac",,"IP", "Status"];
                                for (let i = 0; i  < clientList.length; i++ ) {
                                        if ( clientList[i].mac.length > columnSize[1] )
                                                columnSize[1] =  clientList[i].mac.length;
                                        if ( clientList[i].state.length > columnSize[3] )
                                                columnSize[3] = clientList[i].state.length;
					if ( clientList[i].IP.length > columnSize[2] )
                                                columnSize[2] = clientList[i].IP.length;
					}
                                $("#clients").append("+");
                                for ( let i = 0; i < columnSize[0]; i++) {
                                        $("#clients").append("-");
                                }
                                $("#clients").append("+");
                                for ( let i = 0; i < columnSize[1]; i++) {
                                        $("#clients").append("-");
                                }
                                $("#clients").append("+");
                                for ( let i = 0; i < columnSize[2]; i++) {
                                        $("#clients").append("-");
                                }
                                $("#clients").append("+");
				for ( let i = 0; i < columnSize[3]; i++) {
                                        $("#clients").append("-");
                                }
                                $("#clients").append("+<BR>");
                                $("#clients").append(addSpace("ID", columnSize[0]));
                                $("#clients").append(addSpace("MAC", columnSize[1]));
                                $("#clients").append(addSpace("IP", columnSize[2]));
                                $("#clients").append(addSpace("State", columnSize[3]));
                                $("#clients").append("|<BR>");
                                $("#clients").append("+");
                                for ( let i = 0; i < columnSize[0]; i++) {
                                        $("#clients").append("-");
                                }
                                $("#clients").append("+");
                                for ( let i = 0; i < columnSize[1]; i++) {
                                        $("#clients").append("-");
                                }
                                $("#clients").append("+");
                                for ( let i = 0; i < columnSize[2]; i++) {
                                        $("#clients").append("-");
                                }
                                $("#clients").append("+");
				for ( let i = 0; i < columnSize[3]; i++) {
                                        $("#clients").append("-");
                                }
                                $("#clients").append("+<BR>");
                                for (let i = 0; i  < clientList.length; i++ ) {
                                        $("#clients").append(addSpace(i.toString(),columnSize[0]));
                                        $("#clients").append(addSpace(clientList[i].mac, columnSize[1]));
                                        $("#clients").append(addSpace(clientList[i].IP, columnSize[2]));
                                        $("#clients").append(addSpace(clientList[i].state, columnSize[3]));
                                        $("#clients").append("|");
                                        if ( clientList[i].state == "new" ) {
                                                $("#clients").append("<label id='"+clientList[i].mac+"'> X</label>");
                                                document.getElementById(clientList[i].mac).onclick = function () {
							var Url = '/client/'+clientList[i].mac;
					                $.ajax({
						                type: "GET",
						                contentType: 'application/json',
						                url: Url,
					                	success: function(response){
									updateClients();
								}
							});
                                                };
                                        }
                                        $("#clients").append("<BR>");
                                }
				$("#clients").append("+");
                                for ( let i = 0; i < columnSize[0]; i++) {
                                        $("#clients").append("-");
                                }
                                $("#clients").append("+");
                                for ( let i = 0; i < columnSize[1]; i++) {
                                        $("#clients").append("-");
                                }
                                $("#clients").append("+");
                                for ( let i = 0; i < columnSize[2]; i++) {
                                        $("#clients").append("-");
                                }
				$("#clients").append("+");
                                for ( let i = 0; i < columnSize[3]; i++) {
                                        $("#clients").append("-");
                                }
                                $("#clients").append("+<BR>");
                        }
                }
        });
}
function main(){
	$("#titre").html("Welcome to coreDHCP<br>"+"===================");
	updateClients();
}

