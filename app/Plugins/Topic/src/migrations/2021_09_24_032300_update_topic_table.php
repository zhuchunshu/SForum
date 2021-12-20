<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdateTopicTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('topic', function (Blueprint $table) {
            $table->integer("topping")->comment("置顶")->default(0)->nullable();
            $table->integer('essence')->comment("精华")->default(0)->nullable();
            $table->string("updated_user")->comment("最后更新用户id")->nullable();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('topic', function (Blueprint $table) {
            //
        });
    }
}
