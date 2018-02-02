var serverBase = '/request?url=';
var separator = '/'

function connect() {
    reload();
}

function resetValue() {
    $('#elayout').layout('panel','center').panel('setTitle', '/');
    editor.getSession().setValue('');
    editor.setReadOnly(true);
    $('#footer').html('&nbsp;');
}

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
                    if (autoFormat === 'true') {
                        format(aceMode);
                    }
                    var ttl = 0;
                    if (data.node.ttl) {
                        ttl = data.node.ttl;
                    }
                    changeFooter(ttl, data.node.createdIndex, data.node.modifiedIndex);
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

function changeMode(mode) {
    $('#' + curIconMode).remove();
    editor.getSession().setMode('ace/mode/' + mode);
    curIconMode = 'mode_icon_' + mode
    $('#mode_' + mode).append('<div id="' + curIconMode + '" class="menu-icon icon-ok"></div>');
}