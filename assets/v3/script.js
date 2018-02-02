var separator = '';
var serverBase = '';
var treeMode = Cookies.get('tree-mode');
if (typeof(treeMode) === 'undefined') {
    treeMode = 'list';
}

$.ajax({
    type: 'GET',
    timeout: 5000,
    url:  serverBase + '/separator',
    async: false,
    dataType: 'text',
    success: function(data) {
        separator = data;
    },
    error: function(err) {
        $.messager.alert('Error', $.toJSON(err), 'error');
    }
});

function connect() {
    var status = 'ok';
    $.ajax({
        type: 'POST',
        timeout: 5000,
        url:  serverBase + '/connect',
        data: {'host': etcdBase},
        async: false,
        dataType: 'text',
        success: function(data) {
            if (data === 'ok') {
                console.log('Connect etcd success.');
                //alertMessage('Connect etcd success.');
            }else if (data === 'running') {
                console.log('etcd is running.');
            }else {
                $.messager.alert('Error', data, 'error');
                status = 'error'
            }
        },
        error: function(err) {
            $.messager.alert('Error', $.toJSON(err), 'error');
        }
    });
    
    if (status === 'ok') {
        reload();
    }else {
        resetValue();
        $('#etree').tree('loadData', []);
    }
}

function resetValue() {
    $('#elayout').layout('panel','center').panel('setTitle', separator);
    editor.getSession().setValue('');
    editor.setReadOnly(false);
    $('#footer').html('&nbsp;');
}

function showNode(node) {
    $('#elayout').layout('panel','center').panel('setTitle', node.path);
    editor.getSession().setValue('');
    if (node.dir === false) {
        editor.setReadOnly(false);
        $.ajax({
            type: 'GET',
            timeout: 5000,
            url:  serverBase + '/get',
            data: {'key': node.path},
            async: true,
            dataType: 'json',
            success: function(data) {
                if (data.errorCode) {
                    $('#etree').tree('remove', node.target);
                    resetValue()
                }else {
                    editor.getSession().setValue(data.node.value);
                    /*if (autoFormat === 'true') {
                        format(aceMode);
                    }*/
                    var ttl = 0;
                    if (data.node.ttl) {
                        ttl = data.node.ttl;
                    }
                    changeFooter(ttl, data.node.createdIndex, data.node.modifiedIndex);
                    changeModeBySuffix(node.path);
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
        //editor.setReadOnly(true);
        $('#footer').html('&nbsp;');
        // clear child node
        var children = $('#etree').tree('getChildren', node.target);
        //if (node.state === 'closed' || children.length === 0) {
            
        //}
        var url = '';
        if (treeMode === 'list') {
            url = serverBase + '/get';
        } else {
            url = serverBase + '/getpath';
        }
        $.ajax({
            type: 'GET',
            timeout: 5000,
            url:  url,
            data: {'key': node.path, 'prefix': 'true'},
            async: true,
            dataType: 'json',
            success: function(data) {
                if (data.errorCode) {
                    $.messager.alert('Error',data.errorCode,'error');
                }else {
                    if (data.node.value) {
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
            },
            error: function(err) {
                $.messager.alert('Error',$.toJSON(err),'error');
            }
        });
    }
}

function getNode(n) {
    var text = '';
    if (treeMode === 'list') {
        text = n.key;
    } else {
        var path = n.key.split(separator);
        text = path[path.length - 1];
    }
    var obj = {
        id  :    getId(),
        text:    text,
        dir:     false,
        iconCls: 'icon-text',
        path:    n.key,
        children:[]
    };
    if (n.dir === true) {
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
    var mid = "treeNodeMenu";
    if (treeMode === 'path') {
        mid = 'treeDirMenu';
    } else {
        if (node.dir === true) {
            mid = 'treeDirMenu';
        }
    }
    $('#' + mid).menu('show',{
        left: e.pageX,
        top: e.pageY
    });
}

function saveValue() {
    var node = $('#etree').tree('getSelected');
    $.ajax({
        type: 'PUT',
        timeout: 5000,
        url:  serverBase + '/put',
        data: {'key': node.path, 'value':editor.getValue()},
        async: true,
        dataType: 'json',
        success: function(data) {
            editor.getSession().setValue(data.node.value);
            var ttl = 0;
            if (data.node.ttl) {
                ttl = data.node.ttl;
            }
            changeFooter(ttl, data.node.createdIndex, data.node.modifiedIndex);
            alertMessage('Save success.');
        },
        error: function(err) {
            $.messager.alert('Error',$.toJSON(err),'error');
        }
    });
}

function createNode() {
    var node = $('#etree').tree('getSelected');
    var nodePath = node.path;
    if (nodePath === separator) {
        nodePath = ''
    }
    
    if (treeMode == 'list') {
        if ($('#cnodeForm').form('validate')) {
            var createNodePath = $('#name').textbox('getValue');
            if (!createNodePath.startsWith(separator)) {
                createNodePath= separator + $('#name').textbox('getValue');
            }
            $.ajax({
                type: 'PUT',
                timeout: 5000,
                url:  serverBase + '/put',
                data: {'key':createNodePath,'value':$('#cvalue').textbox().val(),'ttl':$('#ttl').numberbox().val()},
                async: true,
                dataType: 'text',
                success: function(data) {
                    $('#cnode').window('close');
                    var ret = $.evalJSON(data);
                    if (ret.errorCode) {
                        $.messager.alert('Error', ret.cause + " " + ret.message, 'error');
                    }else {
                        alertMessage('Create success.');
                        var newData = [];
                        var obj = {
                                    id  :    getId(),
                                    text:    createNodePath,
                                    state:   $('#dir').combobox('getValue') === 'true'?'closed':'',
                                    dir:     $('#dir').combobox('getValue') === 'true',
                                    iconCls: $('#dir').combobox('getValue') === 'true'?'icon-dir':'icon-text',
                                    path:    createNodePath,
                                    children:[]
                                };
                        var objNode = nodeExist(obj.path);
                        if (objNode === null) {
                            newData.push(obj);
                        
                            $('#etree').tree('append', {
                                parent: node.target,
                                data: newData
                            });
                        }
                        
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
    } else {
        if ($('#cnodeForm').form('validate')) {
            var pathArr = []
            var inputArr = $('#name').textbox('getValue').split(separator)
            for (var i in inputArr) {
                if ($.trim(inputArr[i]) != '') {
                    pathArr.push(inputArr[i])
                }
            }
            
            $.ajax({
                type: 'PUT',
                timeout: 5000,
                url:  serverBase + '/put',
                data: {'key':nodePath + separator + pathArr.join(separator),'value':$('#cvalue').textbox().val(),'ttl':$('#ttl').numberbox().val()},
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
                                    path:    (prePath==separator?(prePath + ''):(prePath + separator)) + pathArr[k],
                                    children:[]
                                };
                            }else {
                                obj = {
                                    id  :    getId(),
                                    text:    pathArr[k],
                                    state:   'closed',
                                    dir:     true,
                                    iconCls: 'icon-dir',
                                    path:    (prePath==separator?(prePath + ''):(prePath + separator)) + pathArr[k],
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
                        
                        $('#etree').tree('update', {
                            target: node.target,
                            iconCls: 'icon-dir'
                        });
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
    
}

function removeNode() {
    var node = $('#etree').tree('getSelected');
    $.messager.confirm('Confirm', 'Remove ' + node.text + '?', function(r){
        if (r){
            $.ajax({
                type: 'POST',
                timeout: 5000,
                url:  serverBase + '/delete',
                data: {'key': node.path, 'dir':node.dir},
                async: true,
                dataType: 'text',
                success: function(data) {
                    resetValue();
                    if (data === 'ok') {
                        alertMessage('Delete success.');
                        
                        var pnode = $('#etree').tree('getParent', node.target);
                        
                        $('#etree').tree('remove', node.target);
                        
                        var isLeaf = $('#etree').tree('isLeaf', pnode.target);
                        if (isLeaf) {
                            $('#etree').tree('update', {
                                target: pnode.target,
                                iconCls: 'icon-text'
                            });
                        }
                    }else {
                        $.messager.alert('Error', data, 'error');
                    }
                },
                error: function(err) {
                    $.messager.alert('Error', $.toJSON(err), 'error');
                }
            });
        }
    });
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

function changeTreeMode() {
    if (treeMode === 'list') {
        treeMode = 'path';
    } else {
        treeMode = 'list';
    }
    Cookies.set('tree-mode', treeMode, {expires: 30});
    connect();
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