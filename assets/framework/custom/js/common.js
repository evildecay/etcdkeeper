/**
 * 时间对象的格式化
 */
Date.prototype.format = function(format) {
	/*
	 * format="yyyy-MM-dd hh:mm:ss";
	 */
	var o = {
		"M+" : this.getMonth() + 1,
		"d+" : this.getDate(),
		"h+" : this.getHours(),
		"m+" : this.getMinutes(),
		"s+" : this.getSeconds(),
		"q+" : Math.floor((this.getMonth() + 3) / 3),
		"S" : this.getMilliseconds()
	};

	if (/(y+)/.test(format)) {
		format = format.replace(RegExp.$1, (this.getFullYear() + "")
				.substr(4 - RegExp.$1.length));
	}

	for ( var k in o) {
		if (new RegExp("(" + k + ")").test(format)) {
			format = format.replace(RegExp.$1, RegExp.$1.length == 1 ? o[k]
					: ("00" + o[k]).substr(("" + o[k]).length));
		}
	}
	return format;
};

/**
 * format date long to HH:ss
 * @param dl
 * @returns {String}
 */
function getTime(dl) {
	var date = new Date(Number(dl));
	var hour = date.getHours();
	var min = date.getMinutes();
	if(hour < 10)
		hour = "0" + hour;
	if(min < 10)
		min = "0" + min;
	return hour + ":" + min;
}

/**
 * format date long to yyyy年MM月dd日
 * @param dl
 * @returns {String}
 */
function getDate(dl) {
	var date = new Date(Number(dl));
	var year = date.getFullYear();
	var month = date.getMonth()+1;
	var day = date.getDate();
	return year+"-"+month+"-"+day;
}

/**
 * get date id format:yyyyMMdd
 * @returns
 */
function getDateId() {
	var date = new Date();
	var year = date.getFullYear();
	var month = date.getMonth()+1;
	var day = date.getDate();
	return year+month+day;
}

/**
 * get date by custom day 
 * @returns yyyy-MM-dd
 */
function getNextDay(date,i) {
	var a = new Date(date);
	a = a.valueOf();
	a = a + i * 24 * 60 * 60 * 1000;
	a = new Date(a);
	//初始化时间
	var year = a.getFullYear();
    var	month = a.getMonth() + 1;
	var day = a.getDate();
	var currentDate = year + "-";
	if (month > 9) {
		currentDate += (month + "-");
	} else {
		currentDate += ("0" + month + "-");
	}
	if (day > 9) {
		currentDate += day;
	} else {
		currentDate += ("0" + day);
	}
	return currentDate;
}

/**
 * format number to 12,345,678
 * @param amount
 * @returns
 */
function CommaFormatted(amount) {
    var delimiter = ","; // replace comma if desired
    amount = new String(amount);
    var a = amount.split('.',2);
    var d = a[1];
    var i = parseInt(a[0]);
    if(isNaN(i)) { return ''; }
    var minus = '';
    if(i < 0) { minus = '-'; }
    i = Math.abs(i);
    var n = new String(i);
    var a = [];
    while(n.length > 3)
    {
        var nn = n.substr(n.length-3);
        a.unshift(nn);
        n = n.substr(0,n.length-3);
    }
    if(n.length > 0) { a.unshift(n); }
    n = a.join(delimiter);
    if(d == null || d.length < 1) { amount = n; }
    else { amount = n + '.' + d; }
    amount = minus + amount;
    return amount;
}