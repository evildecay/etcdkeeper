var aceMode = Cookies.get('ace-mode');
if (typeof(aceMode) === 'undefined') {
    aceMode = 'text';
}
var curIconMode = 'mode_icon_text'
var etcdBase = Cookies.get("etcd-endpoint");
if(typeof(etcdBase) === 'undefined') {
    etcdBase = "127.0.0.1:2379";
}
var tree = [];
var idCount = 0;
var editor = ace.edit('value');

$('#etcdVersion').combobox({
    onChange: changeVersion
});

$(document).ready(function() {
    editor.setTheme('ace/theme/github');
    editor.getSession().setMode('ace/mode/' + aceMode);
    changeMode(aceMode);
    init();
});

function reload() {
    var rootNode = {
        id      : getId(),
        children: [],
        dir     : true,
        path    : separator,
        text    : separator,
        iconCls : 'icon-dir'
    };
    tree = [];
    tree.push(rootNode);
    $('#etree').tree('loadData', tree);
    showNode($('#etree').tree('getRoot'));
    resetValue();
}

function changeMode(mode) {
    aceMode = mode;
    Cookies.set('ace-mode', aceMode, {expires: 30});
    $('#' + curIconMode).remove();
    editor.getSession().setMode('ace/mode/' + aceMode);
    curIconMode = 'mode_icon_' + aceMode;
    $('#mode_' + mode).append('<div id="' + curIconMode + '" class="menu-icon icon-ok"></div>');
    $('#showMode').html(aceMode);
}

function init() {
    $('#etcdAddr').textbox('setValue', etcdBase);
    var t = $('#etree').tree({
        animate:true,
        onClick:showNode,
        onContextMenu:showMenu
    });
}

function changeHost(newValue, oldValue) {
    if (newValue == '') {
        $.messager.alert('Error','ETCD address is empty.','error');
    }
    Cookies.set('etcd-endpoint', newValue, {expires: 30});
    etcdBase = newValue;
    connect();
}

function changeFooter(ttl, cIndex, mIndex) {
    $('#footer').html('<span>TTL&nbsp;:&nbsp;' + ttl + '&nbsp;&nbsp;&nbsp;&nbsp;CreateRevision&nbsp;:&nbsp;' + cIndex + '&nbsp;&nbsp;&nbsp;&nbsp;ModRevision&nbsp;:&nbsp;' + mIndex + '</span><span id="showMode" style="position: absolute;right: 10px;color: #777;">' + aceMode + '</span>');
}

function nodeExist(p) {
    for (var i=0;i<=idCount;i++) {
        var node = $('#etree').tree('find', i);
        if (node != null && node.path == p) {
            return node;
        }
    }
    return null;
}

function selDir(item) {
    if (item.value === 'true') {
        $('#cvalue').textbox('disable','none');
    }else {
        $('#cvalue').textbox('enable','none');
    }
}

function alertMessage(msg) {
    $.messager.show({
        title:'Message',
        msg:msg,
        showType:'slide',
        timeout:1000,
        style:{
            right:'',
            bottom:''
        }
    });
}

function getId() {
    return idCount++;
}

function changeVersion(version) {
    Cookies.set('etcd-version', version, {expires: 30});
    window.location.href = "../" + version
}