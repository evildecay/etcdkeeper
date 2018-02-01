var serverBase = '/request?url=';
var etcdBase = Cookies.get("etcd-endpoint");
if(typeof(etcdBase) === 'undefined') {
    etcdBase = "127.0.0.1:2379";
}
var tree = [];
var idCount = 0;
var editor = ace.edit('value');
var currentMode = 'mode_icon_text'

$(document).ready(function() {
    editor.setTheme('ace/theme/github');
    editor.getSession().setMode('ace/mode/text');
    $('#mode_text').append('<div id="' + currentMode + '" class="menu-icon icon-ok"></div>');
    init();
});

$('#etcdVersion').combobox({
    onChange: changeVersion
})

function init() {
    $('#etcdAddr').textbox('setValue', etcdBase);
    var t = $('#etree').tree({
        animate:true,
        onClick:showNode,
        //lines:true,
        onContextMenu:showMenu
    });
    //queryAll();
    //$("#etree").tree("loadData", tree);
}

function connect(newValue, oldValue) {
    if (newValue == '') {
        $.messager.alert('Error','ETCD address is empty.','error');
    }
    Cookies.set('etcd-endpoint', newValue, {expires: 30});
    etcdBase = newValue;
    reload();
}

function reload() {
    //queryAll();
    var rootNode = {
        id      : getId(),
        children: [],
        dir     : true,
        path    : '/',
        text    : '/',
        iconCls : 'icon-dir'
    };
    
    tree = []
    tree.push(rootNode)
    $('#etree').tree('loadData', tree);
    showNode($('#etree').tree('getRoot'));
    resetValue();
}

function resetValue() {
    $('#elayout').layout('panel','center').panel('setTitle', '/');
    editor.getSession().setValue('');
    editor.setReadOnly(true);
    $('#footer').html('&nbsp;');
}

/*
function queryAll() {
    tree = [];
    $.ajax({
        type: "GET",
        timeout: 10000,
        url:  serverBase + encodeURIComponent(etcdBase + "/?recursive=true&sorted=true"),
        data: {},
        async: false,
        dataType: "json",
        success: function(data) {
            data.node.key = "/";
            loopNodes(data.node, tree);
        },
        error: function(err) {
            $.messager.alert('Error',$.toJSON(err),'error');
        }
    });
}

function loopNodes(node, parent) {
    var curNode = {};
    curNode.id = getId();
    curNode.children = [];
    curNode.dir = false;
    curNode.path = node.key;
    curNode.iconCls = "icon-text";
    if (node.key == "/") {
        curNode.text = "/";
    }else {
        curNode.text = node.key.split("/").pop();
    }
    if (node.dir == true) {
        curNode.state = "closed";
        curNode.dir = true;
        curNode.iconCls = "icon-dir";
        if (node.nodes) {
            for (var i in node.nodes) {
                loopNodes(node.nodes[i], curNode);
            }
        }
    }
    if (node.key == "/") {
        parent.push(curNode);
    }else {
        parent.children.push(curNode);
    }
}
*/

function showNode(node) {
    $('#elayout').layout('panel','center').panel('setTitle', node.path);
    editor.getSession().setValue('');
    if (node.dir == false) {
        editor.setReadOnly(false);
        $.ajax({
            type: 'GET',
            timeout: 5000,
            url:  serverBase + etcdBase + '/v2/keys' + node.path,
            data: '',
            async: true,
            dataType: 'json',
            success: function(data) {
                if (data.errorCode) {
                    $('#etree').tree('remove', node.target);
                    resetValue()
                }else {
                    editor.getSession().setValue(data.node.value);
                    var ttl = 0;
                    if (data.node.ttl) {
                        ttl = data.node.ttl;
                    }
                    $('#footer').html('TTL&nbsp;:&nbsp;' + ttl + '&nbsp;&nbsp;&nbsp;&nbsp;CIndex&nbsp;:&nbsp;' + data.node.createdIndex + '&nbsp;&nbsp;&nbsp;&nbsp;MIndex&nbsp;:&nbsp;' + data.node.modifiedIndex);							
                } 
            },
            error: function(err) {
                $.messager.alert('Error',$.toJSON(err),'error');
            }
        });
    }else {
        if (node.children.length > 0) {
            $('#etree').tree(node.state === 'closed' ? 'expand' : 'collapse', node.target);
        }
        editor.setReadOnly(true);
        $('#footer').html('&nbsp;');
        
        // clear child node
        var children = $('#etree').tree('getChildren', node.target)
        if (node.state == 'closed' || children.length == 0) {
            $.ajax({
                type: 'GET',
                timeout: 5000,
                url:  serverBase + encodeURIComponent(etcdBase + '/v2/keys' + node.path + '?recursive=true&sorted=true'),
                data: '',
                async: true,
                dataType: 'json',
                success: function(data) {
                    if (data.errorCode) {
                        return
                    }else {
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
                },
                error: function(err) {
                    $.messager.alert('Error',$.toJSON(err),'error');
                }
            });
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
    if (node.dir == true) {
        mid = 'treeDirMenu';
    }
    $('#' + mid).menu('show',{
        left: e.pageX,
        top: e.pageY
    });
}

function saveValue() {
    var node = $('#etree').tree('getSelected');
    if (!node.dir) {
        $.ajax({
            type: 'PUT',
            timeout: 5000,
            url:  serverBase + etcdBase + '/v2/keys' + node.path,
            data: {value:editor.getValue()},
            async: true,
            dataType: 'json',
            success: function(data) {
                editor.getSession().setValue(data.node.value);
                var ttl = 0;
                if (data.node.ttl) {
                    ttl = data.node.ttl;
                }
                $('#footer').html('TTL&nbsp;:&nbsp;' + ttl + '&nbsp;&nbsp;&nbsp;&nbsp;CIndex&nbsp;:&nbsp;' + data.node.createdIndex + '&nbsp;&nbsp;&nbsp;&nbsp;MIndex&nbsp;:&nbsp;' + data.node.modifiedIndex);
                alertMessage('Save success.');
            },
            error: function(err) {
                $.messager.alert('Error',$.toJSON(err),'error');
            }
        });
    }
}

function createNode() {
    var node = $('#etree').tree('getSelected');
    var nodePath = node.path;
    if (nodePath == '/') {
        nodePath = ''
    }
    if ($('#cnodeForm').form('validate')) {
        var pathArr = []
        var inputArr = $('#name').textbox('getValue').split('/')
        for (var i in inputArr) {
            if ($.trim(inputArr[i]) != '') {
                pathArr.push(inputArr[i])
            }
        }
        $.ajax({
            type: 'PUT',
            timeout: 5000,
            url:  serverBase + etcdBase + '/v2/keys' + nodePath + '/' + pathArr.join('/'),
            data: {dir:$('#dir').combobox('getValue'),value:$('#cvalue').textbox().val(),ttl:$('#ttl').numberbox().val()},
            async: true,
            dataType: 'text',
            success: function(data) {
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
                        var obj = {};
                        if (k == pathArr.length - 1) {
                            obj = {
                                id  :    getId(),
                                text:    pathArr[k],
                                state:   $('#dir').combobox('getValue') == 'true'?'closed':'',
                                dir:     $('#dir').combobox('getValue') == 'true'?true:false,
                                iconCls: $('#dir').combobox('getValue') == 'true'?'icon-dir':'icon-text',
                                path:    (prePath=='/'?(prePath + ''):(prePath + '/')) + pathArr[k],
                                children:[]
                            };
                        }else {
                            obj = {
                                id  :    getId(),
                                text:    pathArr[k],
                                state:   'closed',
                                dir:     true,
                                iconCls: 'icon-dir',
                                path:    (prePath=='/'?(prePath + ''):(prePath + '/')) + pathArr[k],
                                children:[]
                            };
                        }
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
                    $('#etree').tree('append', {
                        parent: node.target,
                        data: newData
                    });
                }
                $('#cvalue').textbox('enable','none');
                $('#cnodeForm').form('reset');
                $('#ttl').numberbox('setValue', '');
            },
            error: function(err) {
                $.messager.alert('Error', err, 'error');
            }
        });
    }
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

function removeNode() {
    var node = $('#etree').tree('getSelected');
    $.messager.confirm('Confirm', 'Remove ' + node.text + '?', function(r){
        if (r){
            $.ajax({
                type: 'DELETE',
                timeout: 5000,
                url:  serverBase + etcdBase + '/v2/keys' + node.path + '?recursive=true',
                data: {},
                async: true,
                dataType: 'text',
                success: function(data) {
                    resetValue();
                    alertMessage('Delete success.');
                    $('#etree').tree('remove', node.target);
                },
                error: function(err) {
                    $.messager.alert('Error',err,'error');
                }
            });
        }
    });
}

function selDir(item) {
    if (item.value == 'true') {
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

function changeMode(mode) {
    $('#' + currentMode).remove();
    editor.getSession().setMode('ace/mode/' + mode);
    currentMode = 'mode_icon_' + mode
    $('#mode_' + mode).append('<div id="' + currentMode + '" class="menu-icon icon-ok"></div>');
}