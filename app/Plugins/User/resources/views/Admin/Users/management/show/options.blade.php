<div class="hr-text hr-text-left">其他</div>

<div class="row">

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">qq</label>
        <input class="form-control" type="text" disabled value="@if($user->Options->qq) {{$user->Options->qq}} @else 暂无 @endif">
    </div>
    <div class="col-6 col-lg-3">
        <label for="" class="form-label">微信</label>
        <input class="form-control" type="text" disabled value="@if($user->Options->wx) {{$user->Options->wx}} @else 暂无 @endif">
    </div>


    <div class="col-6 col-lg-3">
        <label for="" class="form-label">网站</label>
        <input class="form-control" type="text" disabled value="@if($user->Options->website) {{$user->Options->website}} @else 暂无 @endif">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">对外邮箱</label>
        <input class="form-control" type="text" disabled value="@if($user->Options->email) {{$user->Options->email}} @else 暂无 @endif">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">积分</label>
        <input class="form-control" type="number" disabled value="@if($user->Options->credits){{$user->Options->credits}}@else{{"0"}}@endif">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">金币</label>
        <input class="form-control" type="number" disabled value="@if($user->Options->golds){{$user->Options->golds}}@else{{"0"}}@endif">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">经验</label>
        <input class="form-control" type="number" disabled value="@if($user->Options->exp){{$user->Options->exp}}@else{{"0"}}@endif">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">余额</label>
        <input class="form-control" min="0.01" step="0.01" type="number" disabled value="@if($user->Options->money){{$user->Options->money}}@else{{"0"}}@endif">
    </div>
    <div class="col-6 col-lg-3">
        <label for="" class="form-label">签名</label>
        <textarea rows="4" class="form-control" disabled>{{$user->Options->qianming}}</textarea>
    </div>
</div>