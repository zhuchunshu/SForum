<div class="hr-text hr-text-left">基本</div>
<div class="row">

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">用户名</label>
        <input class="form-control" type="text" disabled value="{{$user->username}}">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">邮箱</label>
        <input class="form-control" type="email" disabled value="{{$user->email}}">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">邮箱验证时间</label>
        <input class="form-control" type="text" disabled value="@if($user->email_ver_time) {{$user->email_ver_time}} @else 未验证 @endif">
    </div>



    <div class="col-6 col-lg-3">
        <label for="" class="form-label">手机号</label>
        <input class="form-control" type="text" disabled value="@if($user->phone){{$user->phone}}@else 暂无 @endif">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">用户组</label>
        <input class="form-control" type="text" disabled value="{{$user->Class->name}}">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">手机验证时间</label>
        <input class="form-control" type="text" disabled value="@if($user->phone_ver_time) {{$user->phone_ver_time}} @else 未验证 @endif">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">注册时间</label>
        <input class="form-control" type="text" disabled value="@if($user->created_at) {{$user->created_at}} @else 暂无 @endif">
    </div>

</div>