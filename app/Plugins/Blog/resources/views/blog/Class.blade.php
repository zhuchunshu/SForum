@extends('Core::app')

@section('title','分类管理 - 我的博客')

@section('content')

    <div class="col-md-12">
        <div class="row row-cards justify-content-center">

            <div class="col-md-6">

                <a href="/blog/class/list" class="card card-link card-active">
                    <div class="card-body text-center">
                        管理
                    </div>
                </a>

            </div>

            <div class="col-md-6">
                <a href="/blog/class/create" class="card card-link card-active">
                    <div class="card-body text-center">
                        创建
                    </div>
                </a>
            </div>

        </div>
    </div>

@endsection