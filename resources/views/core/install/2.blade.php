<x-csrf/>
<input type="hidden" name="next" value="true">
<h3 style="color: red">初始化成功!</h3>
接下来请回到服务器终端重启服务

<h3 style="color: red">重启服务方法:</h3>
<ul>
    <li>回到终端终止当前服务</li>
    <li>重新运行 <code>php CodeFec CodeFec</code></li>
</ul>

<h3 style="color: red">确定重启服务后点击下一步!</h3>
<script>
    $(function(){
        $("#install-img").attr("src","/install/step2.svg")
    })
</script>