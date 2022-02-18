@extends('app')

@section('title','Releases')

@section('content')
    <div class="col-md-12">
        <div class="row row-cards" id="vue-admin-panel-releases">
            <div class="col-md-12" v-if="data">
                <div class="row row-cards">

{{--                    版本信息--}}
                <div class="col-md-9">
                    <div class="border-0 card">
                        <div class="card-body">
                            <h3 class="card-title">版本信息</h3>
                            <ul>
                                <li>版本id: @{{ data.id  }} </li>
                                <li>版本号: @{{ data.tag_name  }} </li>
                                <li>创建时间: @{{ data.created_at }}</li>
                                <li>发布时间: @{{ data.published_at }}</li>
                                <li>版本链接: <a :href="data.html_url">@{{ data.html_url }}</a> </li>
                                <li v-if="data.prerelease"><span class="badge bg-azure">预发布</span></li>
                            </ul>
                            <a :href="data.zipball_url" class="btn btn-blue" style="margin-right:10px">下载zip包</a>
                            <a :href="data.tarball_url" class="btn btn-primary">下载tar.gz包</a>
                        </div>
                    </div>
                </div>
                {{--                    作者信息--}}
                <div class="col-md-3" v-if="author">
                    <div class="border-0 card">
                        <div class="card-body text-center">
                            <div class="mb-3">
                            <span class="avatar avatar-xl avatar-rounded">
                                <img :src="data.author.avatar_url" alt="">
                            </span>
                            </div>
                            <div class="card-title mb-1">@{{ author.name }}</div>
                            <div class="text-muted">@{{ author.login }}</div>
                        </div>
                        <a :href="author.html_url" class="card-btn">关注</a>
                    </div>
                </div>
                </div>
            </div>
            <div class="col-md-12" v-else>
                <div class="empty">
                    <div class="empty-icon"><!-- Download SVG icon from http://tabler-icons.io/i/mood-sad -->
                        <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"></path><circle cx="12" cy="12" r="9"></circle><line x1="9" y1="10" x2="9.01" y2="10"></line><line x1="15" y1="10" x2="15.01" y2="10"></line><path d="M9.5 15.25a3.5 3.5 0 0 1 5 0"></path></svg>
                    </div>
                    <p class="empty-title">No results found</p>
                    <p class="empty-subtitle text-muted">
                        Try adjusting your search or filter to find what you're looking for.
                    </p>

                </div>
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script>var release_id = {{$id}};</script>
    <script src="{{mix('js/admin/index.js')}}"></script>
@endsection