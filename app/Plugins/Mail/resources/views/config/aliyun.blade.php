<div class="mb-3">
    <label class="form-label">阿里云 accessKeyId</label>
    <input name="Aliyun[accessKeyId]" type="text" value="{{get_options('MAIL_Ali_accessKeyId')}}" class="form-control">
</div>
<div class="mb-3">
    <label class="form-label">阿里云 accessKeySecret</label>
    <input name="Aliyun[accessKeySecret]" type="text" value="{{get_options('MAIL_Ali_accessKeySecret')}}" class="form-control">
</div>
<div class="mb-3">
    <label class="form-label">管理控制台中配置的发信地址</label>
    <input name="Aliyun[AccountName]" type="text" value="{{get_options('MAIL_Ali_AccountName')}}" class="form-control">
</div>
<div class="mb-3">
    <label class="form-label">使用管理控制台中配置的回信地址（状态必须是验证通过）。</label>
    <select name="Aliyun[replyToAddress]" class="form-select">
        <option value="开启">开启</option>
        <option value="关闭"  @if(get_options('MAIL_Ali_replyToAddress','开启')==="关闭"){{"selected"}}@endif>关闭</option>
    </select>
</div>
<div class="mb-3">
    <label class="form-label">地址类型</label>
    <select name="Aliyun[AddressType]" class="form-select">
        <option value="0">0</option>
        <option value="1" @if((string)get_options('MAIL_Ali_AddressType')===(string)"1"){{"selected"}}@endif>1</option>
    </select>
    <small>0：为随机账号  ｜ 1：为发信地址</small>
</div>

<div class="mb-3">
    <label class="form-label">发信人昵称</label>
    <input name="Aliyun[FromAlias]" type="text" value="{{get_options('MAIL_Ali_FromAlias')}}" class="form-control">
</div>
<div class="mb-3">
    <label class="form-label">回信地址</label>
    <input name="Aliyun[ReplyAddress]" type="text" value="{{get_options('MAIL_Ali_ReplyAddress')}}" class="form-control">
</div>
<div class="mb-3">
    <label class="form-label">回信地址别称</label>
    <input name="Aliyun[ReplyAddressAlias]" type="text" value="{{get_options('MAIL_Ali_ReplyAddressAlias')}}" class="form-control">
</div>