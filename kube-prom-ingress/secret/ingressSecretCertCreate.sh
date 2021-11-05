#! /bin/bash

kubectl create secret tls dearxuany-com --cert ./dearxuany.com.pem --key ./dearxuany.com.key -n monitoring --kubeconfig ~/.kubeconfig/uat/uat-admin.kubeconfig
kubectl create secret tls qasdearxuany-com-cn --cert ./qasdearxuany.com.cn.pem --key ./qasdearxuany.com.cn.key -n monitoring --kubeconfig ~/.kubeconfig/qas/qas-admin.kubeconfig
kubectl create secret tls devdearxuany-com --cert ./devdearxuany.com.pem --key ./devdearxuany.com.key -n monitoring --kubeconfig ~/.kubeconfig/dev/dev-admin.kubeconfig
