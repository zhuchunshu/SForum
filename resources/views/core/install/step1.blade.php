@extends('core.install')
@section('title','第一步')
@section('content')
    点击下一步进行初始化
@endsection

@section('footer')
    <div class="col-4">
        <div class="progress">
            <div class="progress-bar" style="width: 20%" role="progressbar" aria-valuenow="20"
                 aria-valuemin="0" aria-valuemax="100">
            </div>
        </div>
        <span>20% Complete</span>
    </div>
    <div class="col">
        <div class="btn-list justify-content-end">
            {{--            <a href="#" class="btn btn-link link-secondary">--}}
            {{--                上一步--}}
            {{--            </a>--}}
            <button type="submit" class="btn btn-primary">
                下一步
            </button>
        </div>
    </div>
@endsection