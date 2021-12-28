<x-csrf/>
<input type="hidden" name="next" value="true">
<h3 class="csrd-title">安装完成! 点击下一步锁定安装页面</h3>
<b style="color: red">锁定安装页面后任何人将无法再访问此安装页面</b>
<script>
    $(function(){
        $("#install-img").attr("src","/install/step6.svg")
    })
</script>