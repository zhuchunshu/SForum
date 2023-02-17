<div class="hr-text hr-text-left">基本</div>
<div class="row">

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">用户名</label>
        <input class="form-control" type="text"  name="basis[username]" value="{{$user->username}}">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">邮箱</label>
        <input class="form-control" type="email" name="basis[email]" value="{{$user->email}}">
    </div>


    <div class="col-6 col-lg-3">
        <label for="" class="form-label">手机号</label>
        <input class="form-control" type="text" name="basis[phone]" value="@if($user->phone){{$user->phone}}@else{{"暂无"}}@endif">
    </div>

    <div class="col-6 col-lg-3">
        <label class="form-label">
            用户组
        </label>
        <select name="basis[class_id]" class="form-select">
            @foreach(App\Plugins\User\src\Models\UserClass::query()->get() as $value)
                <option value="{{$value->id}}" @if ($value->id===$user->Class->id){{"selected"}}@endif>{{$value->name}}</option>
            @endforeach
        </select>
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">手机验证时间</label>
        <input class="form-control" type="text" name="basis[phone_ver_time]" value="@if($user->phone_ver_time){{$user->phone_ver_time}}@else{{"未验证"}}@endif">
    </div>


</div>