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
                        $("#semantic").html("");
                        if ( data != null )
                        {
                                var myarray = Object.keys(data);
                                clientList = [];
                                for (let i = 0; i  < myarray.length; i++) {
                                        var newClient = new client();
                                        newClient.mac = data[myarray[i]].MacAddress;
                                        newClient.state = data[myarray[i]].State;
                                        newClient.IP= data[myarray[i]].IP;
                                        newClient.Label= data[myarray[i]].Label;
                                        clientList.push(newClient);
                                }
                                $('#semantic').css("width","80%");
                                $('#semantic').append("<table class=\"ui selectable celled table\" id=\"table1\">" +
                                  "<thead>" +
                                    "<tr><th>ID</th>" +
                                    "<th>Label</th>" +
                                    "<th>Mac</th>" +
                                    "<th>IP</th>" +
                                    "<th>State</th>" +
                                "</tr></thead>");
                                $('#table1').append("<tbody id=\"Table1\">");
                                for (let i = 0; i  < clientList.length; i++ ) {
                                        $('#Table1').append("<tr id=\"Table1_" + i.toString() + "\">");
                                        $("#Table1_"+i.toString()).append("<td data-label=\"ID\" id=\""+ "Table1_"+i.toString()+"_id" +"\">" + i.toString() + "</td>");
                                        $("#Table1_"+i.toString()).append("<td data-label=\"Label\">" +
                                                "<div class=\"ui transparent input\" id=\""+"Table1_"+i.toString()+"_I_" +"\">" +
							"<input type=\"text\" placeholder=\"Search...\" value=\""+ clientList[i].Label +"\">" +
                                                "</div>" +
                                                "</td>");
                                        document.getElementById("Table1_"+i.toString()+"_I_").oninput = function(e) {
                                                var valueChanged = false;
                                                if (e.type=='propertychange') {
                                                        valueChanged = e.originalEvent.propertyName=='value';
                                                } else {
                                                        valueChanged = true;
                                                }
                                                if (valueChanged) {
                                                        /* Code goes here */
                                                        console.log(e.target.value);
                                                }
                                        };
                                        $("#Table1_"+i.toString()).append("<td data-label=\"Mac\">" + clientList[i].mac + "</td>");
                                        $("#Table1_"+i.toString()).append("<td data-label=\"IP\">" + clientList[i].IP + "</td>");
                                        $("#Table1_"+i.toString()).append("<td data-label=\"State\">" + clientList[i].state + "</td>");
                                        $('#Table1_'+i.toString()).append("</tr>");
                                        if ( clientList[i].state == "new" ) {
                                                // Highlight the row
                                                // if click on row then offer to destroy it
                                                $("#Table1_"+i.toString()+"_id").css('cursor', 'pointer');
                                                $('#Table1_'+ i.toString()).addClass("error");
                                                document.getElementById("Table1_"+i.toString()+"_id").onclick = function () {
                                                        var Url = '/client/'+clientList[i].mac;
                                                        // Removing node
                                                        $.ajax({
                                                                type: "GET",
                                                                contentType: 'application/json',
                                                                url: Url,
                                                                success: function(response){
                                                                        updateClients();
                                                                }
                                                        });
                                                };
                                        } else {
                                                        $("#Table1_"+i.toString()+"_id").css('cursor', 'default');
                                        }
                                }
                                $('#semantic').append("</tbody>");
                                $('#semantic').append("</table>");
                        }
                }
        });
}
function main(){
	updateClients();
}

