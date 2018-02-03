var aceMode = Cookies.get('ace-mode');
var curIconMode = 'mode_icon_text'
var editor = ace.edit('value');
var etcdBase = Cookies.get('etcd-endpoint');
var etcdVersion = Cookies.get('etcd-version');
var idCount = 0;
var tree = [];
var separator = '';
var serverBase = '';
var readOnly = false;
var autoFormat = true;

if (typeof(aceMode) === 'undefined') {
    aceMode = 'text';
    Cookies.set('ace-mode', aceMode, {expires: 30});
}
if(typeof(etcdBase) === 'undefined') {
    etcdBase = 'http://etcd:2379';
    Cookies.set('etcd-endpoint', etcdBase, {expires: 30});
}
if(typeof(etcdVersion) === 'undefined') {
    etcdVersion = '3';
    Cookies.set('etcd-version', etcdVersion, {expires: 30});
}

$('#etcdVersion').val(etcdVersion);
$('#etcdVersion').combobox({
    onChange: changeVersion
});

$(document).ready(function() {
    editor.$blockScrolling = Infinity;
    editor.setTheme('ace/theme/github');
    editor.getSession().setMode('ace/mode/' + aceMode);
    $('#etcdAddr').textbox('setValue', etcdBase);
    $('#etcdAddr').textbox({
        onChange:changeHost
    })
    changeMode(aceMode);
    init();
});

function doAjax(type, url, data, dataType, successFunction, errorFunction, async) {
    if (typeof(async) === 'undefined' || async === null) {
        async = true;
    }
    $.ajax({
        type: type,
        timeout: 5000,
        url:  url,
        data: data,
        async: async,
        dataType: dataType,
        success: successFunction,
        error: errorFunction
    });
}

function errorMessage(err) {
    $.messager.alert('Error',$.toJSON(err),'error');
}

function init() {
    var t = $('#etree').tree({
        animate:true,
        onClick:showNode,
        onContextMenu:showMenu
    });
    loadValues();
}

function loadValues() {
    if (etcdVersion === '2') {
        readOnly = true;
        serverBase = '/request?url=';
    } else {
        readOnly = false;
        serverBase = '';
        var url = serverBase + '/separator';
        doAjax('GET', url, '', 'text', changeSeparator, errorMessage, false);
    }
    connect();
}

function connect() {
    var status = 'ok';
    if (etcdVersion === '3') {
        $.ajax({
            type: 'POST',
            timeout: 5000,
            url:  serverBase + '/connect',
            data: {'host': etcdBase},
            async: false,
            dataType: 'text',
            success: function(data) {
                if (data === 'ok' || data === 'running') {
                    console.log('Connect etcd success.');
                } else {
                    $.messager.alert('Error', data, 'error');
                    status = 'error'
                }
            },
            error: function(err) {
                $.messager.alert('Error', $.toJSON(err), 'error');
            }
        });
    }
    
    if (status === 'ok') {
        reload();
    } else {
        resetValue();
        $('#etree').tree('loadData', []);
    }
}

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

function resetValue() {
    $('#elayout').layout('panel','center').panel('setTitle', separator);
    editor.getSession().setValue('');
    editor.setReadOnly(readOnly);
    $('#footer').html('&nbsp;');
}

function showNodeOk(data, node) {
    if (data.errorCode) {
        $('#etree').tree('remove', node.target);
        resetValue()
    }else {
        editor.getSession().setValue(data.node.value);
        if (autoFormat) {
            format(aceMode);
        }
        var ttl = 0;
        if (data.node.ttl) {
            ttl = data.node.ttl;
        }
        changeFooter(ttl, data.node.createdIndex, data.node.modifiedIndex);
        if (etcdVersion === '3') {
            changeModeBySuffix(node.path);
        }
    }
}

function showNodeOk2(data, node) {
    var children = $('#etree').tree('getChildren', node.target);
    if (data.errorCode) {
        $.messager.alert('Error',data.errorCode,'error');
    }else {
        if (etcdVersion === '3' && data.node.value) {
            editor.getSession().setValue(data.node.value);
            changeFooter(data.node.ttl, data.node.createdIndex, data.node.modifiedIndex);
            changeModeBySuffix(node.path);
        }
        var arr = [];
        
        if (data.node.nodes) {									
            // refresh child node
            for (var i in data.node.nodes) {
                var newData = getNode(data.node.nodes[i]);	
                arr.push(newData);
            }
            $('#etree').tree('append', {
                parent: node.target,
                data: arr
            });
        }
        
        for(var n in children) {
            $('#etree').tree('remove', children[n].target);
        }
    } 
}

function showNode(node) {
    $('#elayout').layout('panel','center').panel('setTitle', node.path);
    editor.getSession().setValue('');
    if (node.dir == false) {
        editor.setReadOnly(false);
        var url = '';
        var data;
        if (etcdVersion === '2'){
            url = serverBase + etcdBase + '/v2/keys' + node.path;
            data = '';
        } else {
            url = serverBase + '/get';
            data = {'key': node.path};
        }
        doAjax('GET', url, data, 'json', function(data){showNodeOk(data, node)}, errorMessage);
    } else {
        if (node.children.length > 0) {
            $('#etree').tree(node.state === 'closed' ? 'expand' : 'collapse', node.target);
        }
        editor.setReadOnly(readOnly);
        $('#footer').html('&nbsp;');
        
        // clear child node
        var children = $('#etree').tree('getChildren', node.target);
        var url = '';
        var data;
        if (etcdVersion === '2'){
            url = serverBase + encodeURIComponent(etcdBase + '/v2/keys' + node.path + '?recursive=true&sorted=true');
            data = '';
        } else {
            url = serverBase + '/getpath';
            data = {'key': node.path, 'prefix': 'true'};
        }
        if (node.state == 'closed' || children.length == 0) {
            doAjax('GET', url, data, 'json', function(data){showNodeOk2(data, node)}, errorMessage);
        }
    }
}

function getNode(n) {
    var path = n.key.split('/');
    var obj = {
        id  :    getId(),
        text:    path[path.length - 1],
        dir:     false,
        iconCls: 'icon-text',
        path:    n.key,
        children:[]
    };
    if (n.dir == true) {
        obj.state = 'closed';
        obj.dir = true;
        obj.iconCls = 'icon-dir';
        if (n.nodes) {
            for (var i in n.nodes) {
                var rn = getNode(n.nodes[i]);
                obj.children.push(rn);
            }
        }
    }
    return obj
}

function showMenu(e, node) {
    e.preventDefault();
    $('#etree').tree('select',node.target);
    var mid = 'treeNodeMenu';
    if (etcdVersion === '3' || node.dir) {
        mid = 'treeDirMenu';
    }
    $('#' + mid).menu('show',{
        left: e.pageX,
        top: e.pageY
    });
}

function saveValueOk(data) {
    editor.getSession().setValue(data.node.value);
    var ttl = 0;
    if (data.node.ttl) {
        ttl = data.node.ttl;
    }
    $('#footer').html('TTL&nbsp;:&nbsp;' + ttl + '&nbsp;&nbsp;&nbsp;&nbsp;CIndex&nbsp;:&nbsp;' + data.node.createdIndex + '&nbsp;&nbsp;&nbsp;&nbsp;MIndex&nbsp;:&nbsp;' + data.node.modifiedIndex);
    alertMessage('Save success.');
}

function saveValue() {
    var node = $('#etree').tree('getSelected');
    if (!node.dir) {
        var url = '';
        var data;
        if (etcdVersion === '2'){
            url = serverBase + etcdBase + '/v2/keys' + node.path;
            data = {value:editor.getValue()};
        } else {
            url = serverBase + '/put';
            data = {'key': node.path, 'value':editor.getValue()};
        }
        doAjax('PUT', url, data, 'json', saveValueOk, errorMessage);
    }
}

function createNodeOk(data, node, pathArr) {
    $('#cnode').window('close');
    var ret = $.evalJSON(data);
    if (ret.errorCode) {
        $.messager.alert('Error', ret.cause + ' ' + ret.message, 'error');
    }else {
        alertMessage('Create success.');
        var newData = [];
        var preObj = {};
        var prePath = node.path;
        for (var k in pathArr) {
            var state = 'closed';
            var dir = true;
            var iconCls = 'icon-dir';
            if (k == pathArr.length - 1) {
                state = $('#dir').combobox('getValue') == 'true'?'closed':'';
                dir = $('#dir').combobox('getValue') == 'true'?true:false;
                iconCls = $('#dir').combobox('getValue') == 'true'?'icon-dir':'icon-text';
            }
            var obj = {
                id  :    getId(),
                text:    pathArr[k],
                state:   state,
                dir:     dir,
                iconCls: iconCls,
                path:    (prePath==separator?(prePath + ''):(prePath + separator)) + pathArr[k],
                children:[]
            };

            var objNode = nodeExist(obj.path)
            if (objNode != null) {
                node = objNode;
                prePath = node.path;
                continue;
            }
            if (newData.length == 0) {
                newData.push(obj);
            }else {
                preObj.children.push(obj);
            }
            preObj = obj;
            prePath = obj.path;
        }
        
        if (etcdVersion === '3') {
            $('#etree').tree('update', {
                target: node.target,
                iconCls: 'icon-dir'
            });
        }
        $('#etree').tree('append', {
            parent: node.target,
            data: newData
        });
    }
    
    $('#cvalue').textbox('enable','none');
    $('#cnodeForm').form('reset');
    $('#ttl').numberbox('setValue', '');
}

function createNode() {
    var node = $('#etree').tree('getSelected');
    var nodePath = node.path;
    if (nodePath === separator) {
        nodePath = ''
    }
    if ($('#cnodeForm').form('validate')) {
        var pathArr = []
        var inputArr = $('#name').textbox('getValue').split(separator)
        for (var i in inputArr) {
            if ($.trim(inputArr[i]) != '') {
                pathArr.push(inputArr[i])
            }
        }
        var url = '';
        var data;
        if (etcdVersion === '2'){
            url = serverBase + etcdBase + '/v2/keys' + nodePath + '/' + pathArr.join('/');
            data = {dir:$('#dir').combobox('getValue'),value:$('#cvalue').textbox().val(),ttl:$('#ttl').numberbox().val()};
        } else {
            url = serverBase + '/put';
            data = {'key':nodePath + separator + pathArr.join(separator),'value':$('#cvalue').textbox().val(),'ttl':$('#ttl').numberbox().val()};
        }
        doAjax('PUT', url, data, 'text', function(data){createNodeOk(data, node, pathArr)}, errorMessage);
    }
}

function removeNodeOk(data) {
    var node = $('#etree').tree('getSelected');
    resetValue();
    if (etcdVersion === '2') {
        alertMessage('Delete success.');
        $('#etree').tree('remove', node.target);
    } else {
        if (data === 'ok') {
            alertMessage('Delete success.');
            $('#etree').tree('remove', node.target);
            
            var pnode = $('#etree').tree('getParent', node.target);
            if (pnode) {
                var isLeaf = $('#etree').tree('isLeaf', pnode.target);
                if (isLeaf) {
                    $('#etree').tree('update', {
                        target: pnode.target,
                        iconCls: 'icon-text'
                    });
                }
            }
        } else {
            $.messager.alert('Error', data, 'error');
        }
    }
}

function removeNode() {
    var node = $('#etree').tree('getSelected');
    $.messager.confirm('Confirm', 'Remove ' + node.text + '?', function(r){
        if (r){
            var url = '';
            var data;
            if (etcdVersion === '2') {
                url = serverBase + etcdBase + '/v2/keys' + node.path + '?recursive=true';
                doAjax('DELETE', url, data, 'text', removeNodeOk, errorMessage);
            } else {
                url = serverBase + '/delete';
                data = {'key': node.path, 'dir':node.dir};
                doAjax('POST', url, data, 'text', removeNodeOk, errorMessage);
            }
        }
    });
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
    etcdVersion = version;
    Cookies.set('etcd-version', etcdVersion, {expires: 30});
    loadValues();
}

function changeSeparator(data) {
    separator = data;
}

function format(type) {
    if (type === 'json') {
        val = JSON.parse(editor.getValue());
        editor.setValue(JSON.stringify(val, null, 4));
        editor.getSession().setMode('ace/mode/' + 'json');
        editor.clearSelection();
        editor.navigateFileStart();
    }
}

function changeModeBySuffix(path) {
    var a = path.split(separator);
    var tokens = a.slice(a.length-1,a.lenght)[0].split('.');
    if (tokens.length < 2) {
        return
    }
    var mode = tokens[tokens.length-1]
    var modes = $('#modeMenu').children();
    for (var i=0;i<modes.length;i++) {
        m = modes[i].innerText;
        if (mode === m) {
            changeMode(m);
            return
        }
    }
}