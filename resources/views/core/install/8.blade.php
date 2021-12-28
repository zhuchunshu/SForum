<x-csrf/>
<input type="hidden" name="next" value="true">
<h3 style="color: red">创建管理员用户</h3>
<div class="mb-3">
    <label class="form-label">管理员邮箱</label>
    <input name="email" type="email" class="form-control" autocomplete="off" required>
</div>
<div class="mb-3">
    <label class="form-label">管理员用户名</label>
    <input name="username" type="text" class="form-control" autocomplete="off" required>
</div>
<div class="mb-3">
    <label class="form-label">管理员密码</label>
    <input name="password" type="password" class="form-control" autocomplete="off" required>
</div>
<script>
    $(function(){
        $("#install-img").attr("src","/install/step5.svg")
    })
</script>