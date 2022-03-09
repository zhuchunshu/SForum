@extends('Core::app')

@section('title','分类管理 - 我的博客')

@section('content')

    <div class="col-md-12">
        <div class="row row-cards justify-content-center">
            <div class="col-md-10">
                <div class="border-0 card">
                    <div class="card-body">
                        <div class="row">
                            <div class="col">
                                <h3 class="card-title">
                                    @if(request()->input('parent_id')) <a href="./list"><svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-arrow-back-up" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <path d="M9 13l-4 -4l4 -4m-4 4h11a4 4 0 0 1 0 8h-1"></path>
                                        </svg></a> @endif分类管理</h3>
                            </div>
                            <div class="col-auto">
                                <h3 class="card-title">
                                    <a href="/blog/class/create">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-plus" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <line x1="12" y1="5" x2="12" y2="19"></line>
                                            <line x1="5" y1="12" x2="19" y2="12"></line>
                                        </svg>
                                    </a>
                                </h3>
                            </div>
                        </div>

                        <div class="table-responsive" id="vue-blog-class-list">
                            <table
                                    class="table table-vcenter table-nowrap">
                                <thead>
                                <tr>
                                    <th>ID</th>
                                    <th>名称</th>
                                    <th>token</th>
                                    <th>父分类</th>
                                    <th></th>
                                    <th class="w-1"></th>
                                </tr>
                                </thead>
                                <tbody>
                                @if($list->count())
                                    @foreach($list as $value)
                                        <tr>
                                            <td>{{$value->id}}</td>
                                            <td>{{$value->name}}</td>
                                            <td>{{$value->token}}</td>
                                            <td>
                                                @if($value->parent_id)
                                                    <a href="?parent_id={{$value->parent_id}}">{{$value->parent->name}}</a>
                                                @else
                                                    暂无
                                                @endif
                                            </td>
                                            <td><a href="/blog/class/{{$value->token}}/edit">修改</a></td>
                                            <td><a href="#" @@click="remove('{{$value->token}}')">删除</a></td>
                                        </tr>
                                    @endforeach
                                @else
                                    <tr>
                                        <td>暂无</td>
                                        <td>暂无</td>
                                        <td>暂无</td>
                                        <td>暂无</td>
                                        <td>暂无</td>
                                        <td>暂无</td>
                                    </tr>
                                @endif
                                </tbody>
                            </table>
                        </div>
                    </div>
                    {!! make_page($list) !!}
                </div>
            </div>
        </div>
    </div>

@endsection

@section('scripts')
    <script src="{{file_hash("plugins/Blog/js/class.js")}}"></script>
@endsection