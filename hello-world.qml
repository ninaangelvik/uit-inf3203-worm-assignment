import QtQuick 2.0


Rectangle {
	id: root

    color: "white"
	width: 300
	height: 300

    Text {
            text: ctrl.message

            Component.onCompleted: {
                x = parent.width/2 - width/2
                y = parent.height/2 - height/2
            }

            color: "black"
            font.bold: true
            font.pointSize: 20
        }
}
    
