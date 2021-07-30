@extends('core.install')
@section('title','第四步')
@section('content')
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

@endsection

@section('footer')
    <div class="col-4">
        <div class="progress">
            <div class="progress-bar" style="width: 70%" role="progressbar" aria-valuenow="70"
                 aria-valuemin="0" aria-valuemax="100">
            </div>
        </div>
        <span>70% Complete</span>
    </div>
    <div class="col">
        <div class="btn-list justify-content-end">
                        <a href="/install?step=3" class="btn btn-link link-secondary">
                            上一步
                        </a>
            <button type="submit" class="btn btn-primary">
                下一步
            </button>
        </div>
    </div>
@endsection

@section('action', '/install?step=4')

@section('hr','Redis配置')

@section('img','/install/step4.svg')