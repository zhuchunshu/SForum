@extends('core.install')
@section('title','第二步')
@section('content')
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
@endsection

@section('footer')
    <div class="col-4">
        <div class="progress">
            <div class="progress-bar" style="width: 40%" role="progressbar" aria-valuenow="40"
                 aria-valuemin="0" aria-valuemax="100">
            </div>
        </div>
        <span>40% Complete</span>
    </div>
    <div class="col">
        <div class="btn-list justify-content-end">
            <a href="/install?step=1" class="btn btn-link link-secondary">
                上一步
            </a>
            <button type="submit" class="btn btn-primary">
                下一步
            </button>
        </div>
    </div>
@endsection

@section('action', '/install?step=2')

@section('img','/install/step2.svg')