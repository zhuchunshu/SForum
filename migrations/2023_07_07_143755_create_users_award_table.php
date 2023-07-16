<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreateUsersAwardTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('users_award', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->string('name', 50)->comment('奖励名称');
            $table->bigInteger('user_id')->comment('用户id');
            $table->timestampsTz();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('users_award');
    }
}
