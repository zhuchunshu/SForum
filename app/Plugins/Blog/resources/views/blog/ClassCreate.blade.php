@extends('Core::app')

@section('title','创建分类 - 我的博客')

@section('content')

    <div class="col-md-12">
        <div class="row row-cards justify-content-center">
            <div class="col-md-10">
                <div class="border-0 card">
                    <div class="card-body">
                        <div class="row">
                            <div class="col">
                                <h3 class="card-title">创建分类</h3>
                            </div>
                            <div class="col-auto">
                                <h3 class="card-title">
                                    <a href="./list">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-list" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <line x1="9" y1="6" x2="20" y2="6"></line>
                                            <line x1="9" y1="12" x2="20" y2="12"></line>
                                            <line x1="9" y1="18" x2="20" y2="18"></line>
                                            <line x1="5" y1="6" x2="5" y2="6.01"></line>
                                            <line x1="5" y1="12" x2="5" y2="12.01"></line>
                                            <line x1="5" y1="18" x2="5" y2="18.01"></line>
                                        </svg>
                                    </a>
                                </h3>
                            </div>
                        </div>
                        <form action="" method="post">
                            <x-csrf/>
                            <div class="mb-3">
                                <label for="" class="form-label">
                                    父分类
                                </label>
                                <select name="class_id" class="form-select">
                                    @if(count($classList))
                                        <option value="none">暂不选择</option>
                                        @foreach($classList as $value)
                                            <option value="{{$value->id}}">{{$value->name}}</option>
                                        @endforeach
                                    @else
                                        <option value="none">暂无</option>
                                    @endif
                                </select>
                            </div>
                            <div class="mb-3">
                                <label for="" class="form-label">
                                    分类名
                                </label>
                                <input type="text" name="name" class="form-control" required>
                            </div>
                            <div class="mb-3">
                                <button class="btn btn-primary">提交</button>
                            </div>
                        </form>
                    </div>
                </div>
            </div>
        </div>
    </div>

@endsection

