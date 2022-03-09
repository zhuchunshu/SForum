@extends('Core::app')

@section('title','写文章 - 博客')

@section('content')

    <div class="col-md-12">
        <div class="row row-cards justify-content-center">
            <div class="col-md-10">
                <div class="border-0 card">
                    <div class="card-body" id="vue-blog-article-create">
                        <h3 class="card-title">写文章</h3>
                        <form @@submit.prevent="submit">
                            <div class="mb-3">
                                <label for="" class="form-label">标题</label>
                                <input type="text" v-model="title" class="form-control">
                            </div>
                            <div class="mb-3">
                                <label for="" class="form-label">分类</label>
                                <select name="" v-model="class_id" class="form-select">
                                    <option v-for="option in classList"  :value="option.value">
                                        @{{ option . text }}
                                    </option>
                                </select>
                            </div>
                            <div class="mb-3">
                                <label for="" class="form-label">正文</label>
                                <div id="content-vditor"></div>
                            </div>
                            <button class="btn btn-primary">发布</button>
                        </form>
                    </div>
                </div>
            </div>
        </div>
    </div>

@endsection

@section('scripts')
    <script src="{{file_hash('plugins/Blog/js/article.js')}}"></script>
@endsection
@section('headers')
    <link rel="stylesheet" href="{{file_hash('plugins/Blog/css/article.css')}}">
@endsection