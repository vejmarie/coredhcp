// Copyright 2024 Hewlett Packard Development LP.

var clientList = [];
var firmwareList = [];

function client() {
    this.mac = null;
    this.state = null;
}
function firmware() {
        this.version = null;
        this.date = null;
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
unction updateFirmwares() {
        var Url = '/firmwares';
                $.ajax({
                type: "GET",
                contentType: 'application/json',
                url: Url,
                success: function(response){
                        console.log(response);
                        $("#firmwares").html("");
                        $('#firmwares').css("width","80%");
                        $('#firmwares').append( "<br>Firmware<br><br><div class=\"ui grid\">" +
                                "<div class=\"fourteen wide column\">" +
                                  "<table class=\"ui selectable celled table\" id=\"table_images\">" +
                                  "<thead>" +
                                    "<tr><th>ID</th>" +
                                    "<th>Version</th>" +
                                    "<th>Date</th>" +
                                  "</tr></thead></table>" +
                                "</div>" +
                                "<div class=\"two wide column\">" +
                                     "<img class=\"ui medium rounded image\" src=\"/images/cpu.png\">" +
                                "</div></div>");
                        var data = JSON.parse(response);

                        if ( data != null )
                        {
                                var myarray = Object.keys(data);
                                firmwareList = [];
                                for (let i = 0; i  < myarray.length; i++) {
                                        var newFirmware = new firmware();
                                        newFirmware.version = data[myarray[i]].Version;
                                        newFirmware.date = data[myarray[i]].Date;
					ewFirmware.default = data[myarray[i]].Default;
                                        firmwareList.push(newFirmware);
                                }
                                for (let i = 0; i  < firmwareList.length; i++ ) {
                                        $('#TableFirmware').append("<tr id=\"TableFirmware_" + i.toString() + "\">");
                                        $("#TableFirmware_"+i.toString()).append("<td data-label=\"Default\">"+
                                                "<center><div class='ui fitted checkbox' id=\""+ "TableFirmware_"+i.toString()+"_checked" +"\">" +
                                                  "<input type='checkbox' name='example' id=\"" + "TableFirmware_"+i.toString()+"_checkbox" + "\">" +
                                                  "<label></label>" +
                                                "</div></center>" +
                                                "</td>");

					if ( firmwareList[i].default ) {
                                                $("#TableFirmware_" +i.toString()+"_checkbox").prop( "checked", true );
                                                $("#TableFirmware_" +i.toString()+"_checkbox").attr("disabled", "disabled");
                                        }

                                        // oncheck we have to remove all other checkbox
                                        console.log("#TableFirmware_" +i.toString()+"_checkbox");
                                        document.getElementById("TableFirmware_" +i.toString()+"_checkbox").oninput = function(e) {
                                                console.log("CHANGEMENT");
                                                if ( this.checked ) {
                                                        console.log("Checked");
                                                        $("#TableFirmware_" +i.toString()+"_checkbox").attr("disabled", "disabled");
                                                        for (let j = 0 ; j < firmwareList.length; j++ ) {
                                                                if ( j != i ) {
                                                                        $("#TableFirmware_" +j.toString()+"_checkbox").removeAttr("disabled");
                                                                        $("#TableFirmware_" +j.toString()+"_checkbox").prop("checked", false);
                                                                }
                                                        }
							// We must set our new version as the default
                                                        var data = {
                                                            "Version": "",
                                                        };
                                                        data.Version = firmwareList[i].version;
                                                        var jsonString = JSON.stringify(data);
                                                        $.post("default_firmware", jsonString, function(result){
                                                        });
                                                } else
                                                        console.log("unchecked");
                                        };

                                        $("#TableFirmware_"+i.toString()).append("<td data-label=\"ID\" id=\""+ "TableFirmware_"+i.toString()+"_id" +"\">" + i.toString() + "</td>");
                                        $("#TableFirmware_"+i.toString()).append("<td data-label=\"Version\" id=\""+ "TableFirmware_"+i.toString()+"_version" +"\">"
                                                                           + firmwareList[i].version + "</td>");
                                        $("#TableFirmware_"+i.toString()).append("<td data-label=\"Date\" id=\""+ "TableFirmware_"+i.toString()+"_date" +"\">"
                                                                           + firmwareList[i].date + "</td>");
                                        $('#TableFirmware').append("</tr>");
                                }
                        }





                        var dropZone = document.getElementById('firmwareDrop');

                        var startUploadfirmware = function(files) {
                                var formData = new FormData();
                                for(var i = 0; i < files.length; i++){
                                        var file = files[i];
                                        formData.append('name', file.name);
                                        formData.append('fichier', file);
                                        console.log(file);
                                }
                                console.log(formData);
                                var xhr = new XMLHttpRequest();
                                xhr.open('POST', window.location+'upload_firmware/', true);

                                xhr.onload = function () {
                                  if (xhr.status === 200) {
                                    // File(s) uploaded
                                    updateFirmwares();
                                  } else {
                                    alert('Something went wrong uploading the file.');
                                  }
                                };
                                xhr.upload.addEventListener('progress', function(e) {
                                        var percent = e.loaded / e.total * 100;
                                        $('#uploadProgress').progress({ percent: Math.floor(percent) });
                                        }, false);

                                xhr.send(formData);
                        }



                        dropZone.ondrop = function(e) {
                                e.preventDefault();
                                console.log("New firmware drop");
                                startUploadfirmware(e.dataTransfer.files);
                        }
                        dropZone.ondragover = function() {
                                $('#dropImage').css("opacity", "0.2");
                                console.log("File coming");
                                return false;
                        }

                        dropZone.ondragleave = function() {
                                $('#dropImage').css("opacity", "1.0");
                                console.log("File leaving");
                                return false;
                        }

                        $('#uploadProgress').progress({ percent: 0 });

                        }
                });

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
                                                        // We have to send the new data through a POST call
                                                        var data = {
                                                            "Mac": "",
                                                            "Label": ""
                                                        };
                                                        data.Mac = clientList[i].mac;
                                                        data.Label = e.target.value;
                                                        var jsonString = JSON.stringify(data);
                                                        $.post("label", jsonString);
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
	updateFirmwares();
	updateClients();
}

