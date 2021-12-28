<x-csrf/>
<input type="hidden" name="next" value="true">
<h3 style="color: red">网站信息配置成功!</h3>
重启服务后生效(可暂时不重启)

<h3 style="color: red">重启服务方法:</h3>
<ul>
    <li>回到终端终止当前服务</li>
    <li>重新运行 <code>php CodeFec CodeFec</code></li>
</ul>

<h3 style="color: red">接下来配置数据库信息!</h3>
<div class="mb-3">
    <label class="form-label">数据库地址</label>
    <input name="DB_HOST" value="{{ env('DB_HOST') }}" type="text" class="form-control" autocomplete="off"
           required>
</div>
<div class="mb-3">
    <label class="form-label">数据库名</label>
    <input name="DB_DATABASE" value="{{ env('DB_DATABASE') }}" type="text" class="form-control" autocomplete="off"
           required>
</div>
<div class="mb-3">
    <label class="form-label">数据库用户名</label>
    <input name="DB_USERNAME" value="{{ env('DB_USERNAME') }}" type="text" class="form-control" autocomplete="off"
           required>
</div>
<div class="mb-3">
    <label class="form-label">数据库密码</label>
    <input name="DB_PASSWORD" value="{{ env('DB_PASSWORD') }}" type="password" class="form-control" autocomplete="off"
           required>
</div>
<script>
    $(function(){
        $("#install-img").attr("src","/install/step3.svg")
    })
</script>