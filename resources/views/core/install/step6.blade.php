@extends('core.install')
@section('title','第五步')
@section('content')
    <h3 class="csrd-title">安装完成! 点击下一步锁定安装页面</h3>
    <b style="color: red">锁定安装页面后任何人将无法再访问此安装页面</b>
@endsection

@section('footer')
    <div class="col-4">
        <div class="progress">
            <div class="progress-bar" style="width: 100%" role="progressbar" aria-valuenow="100"
                 aria-valuemin="0" aria-valuemax="100">
            </div>
        </div>
        <span>100% Complete</span>
    </div>
    <div class="col">
        <div class="btn-list justify-content-end">
            <a href="/install?step=5" class="btn btn-link link-secondary">
                上一步
            </a>
            <button type="submit" class="btn btn-primary">
                下一步
            </button>
        </div>
    </div>
@endsection

@section('action', '/install?step=6')

@section('hr','安装完成')
@section('img','/install/step6.svg')