<h3 style="color: red">腾讯云SMS配置 (Qcloud)</h3>
<div class="row">
    <div class="col-2">
        <label class="form-label">SDK APP ID</label>
        <input type="text" v-model="data.sms_qcloud_sdk_app_id" class="form-control">
    </div>
    <div class="col-2">
        <label class="form-label">SECRET ID</label>
        <input type="text" v-model="data.sms_qcloud_secret_id" class="form-control">
    </div>
    <div class="col-2">
        <label class="form-label">SECRET KEY</label>
        <input type="text" v-model="data.sms_qcloud_secret_key" class="form-control">
    </div>
    <div class="col-3">
        <label class="form-label">短信签名</label>
        <input type="text" v-model="data.sms_qcloud_sign_name" class="form-control">
    </div>
    <div class="col-3">
        <label class="form-label">模板ID</label>
        <input type="text" v-model="data.sms_qcloud_template" class="form-control">
    </div>
</div>