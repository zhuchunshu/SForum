<x-csrf/>
<input type="hidden" name="next" value="true">
<h3 style="color: red">配置网站信息</h3>
<div class="mb-3">
    <label class="form-label">网站名称</label>
    <input value="{{ env("APP_NAME") }}" name="name" type="text" class="form-control" autocomplete="off" placeholder="super-forum" required>
</div>
<div class="mb-3">
    <label class="form-label">网站域名</label>
    <input value="{{ env("APP_DOMAIN") }}" name="domain" type="text" class="form-control" autocomplete="off" placeholder="domain.com" required>
</div>
<div class="mb-3">
    <label class="form-label">协议</label>
    @if(env("APP_SSL"))
        <input value="https" name="ssl" type="text" class="form-control" autocomplete="off" placeholder="http(https)" required>
    @else
        <input value="http" name="ssl" type="text" class="form-control" autocomplete="off" placeholder="http(https)" required>
    @endif
</div>
<script>
    $(function(){
        $("#install-img").attr("src","/install/step2.svg")
    })
</script>