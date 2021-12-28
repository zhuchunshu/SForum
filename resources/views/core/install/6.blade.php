<x-csrf/>
<input type="hidden" name="next" value="true">
<h3 style="color: red">配置Redis</h3>
<div class="mb-3">
    <label class="form-label">Redis地址</label>
    <input name="REDIS_HOST" value="{{env("REDIS_HOST")}}" type="text" class="form-control" autocomplete="off" required>
</div>
<div class="mb-3">
    <label class="form-label">Redis密码</label>
    <input name="REDIS_AUTH" value="{{env("REDIS_AUTH")}}" type="text" class="form-control" autocomplete="off">
    <small>默认为空</small>
</div>
<div class="mb-3">
    <label class="form-label">Redis端口</label>
    <input name="REDIS_PORT" value="{{env("REDIS_PORT")}}" type="text" class="form-control" autocomplete="off">
</div>
<script>
    $(function(){
        $("#install-img").attr("src","/install/step4.svg")
    })
</script>