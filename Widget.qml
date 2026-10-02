import QtQuick
import Quickshell.Io
import qs.Commons
import qs.Ui
BarWidget {
  id: root
  moduleName: "hg.tempo"
  readonly property string script: Qt.resolvedUrl("bin/tempo").toString().replace(/^file:\/\//, "")
  property var state: ({})
  property double now: Date.now() / 1000
  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight
  function refresh() { if (!status.running) status.running = true }
  Process {
    id: status
    command: [root.script, "status"]
    stdout: StdioCollector { waitForEnd: true; onStreamFinished: {
      try { root.state = JSON.parse(text) } catch(e) {}
    }}
  }
  Timer { interval: 30000; running: true; repeat: true; onTriggered: root.refresh() }
  Timer { interval: 1000; running: true; repeat: true; onTriggered: root.now = Date.now()/1000 }
  Component.onCompleted: refresh()
  WidgetButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    active: !!root.state.running
    text: {
      if (!root.state.running) return "◷ Tempo"
      var seconds = Math.max(0, Math.floor(root.now - root.state.start))
      return "▶ " + Math.floor(seconds/3600).toString().padStart(2,"0") + ":" + Math.floor(seconds%3600/60).toString().padStart(2,"0")
    }
    tooltipText: root.state.error || (root.state.running ? root.state.description : "Tempo · open time dashboard")
    onPressed: if (root.bar) root.bar.run(root.script + "-window")
  }
}
