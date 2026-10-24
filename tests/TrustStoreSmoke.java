import java.io.FileInputStream;
import java.security.KeyStore;
import java.security.cert.CertificateException;
import java.security.cert.CertificateFactory;
import java.security.cert.X509Certificate;
import javax.net.ssl.TrustManager;
import javax.net.ssl.TrustManagerFactory;
import javax.net.ssl.X509TrustManager;

// Run in a disposable image after adding a private CA with update-ca-certificates.
public class TrustStoreSmoke {
    public static void main(String[] args) throws Exception {
        if (args.length != 15) throw new IllegalArgumentException("Expected a private CA and fourteen incident chains");
        TrustManagerFactory factory = TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm());
        factory.init((KeyStore) null);
        X509TrustManager manager = null;
        for (TrustManager candidate : factory.getTrustManagers()) {
            if (candidate instanceof X509TrustManager) manager = (X509TrustManager) candidate;
        }
        if (manager == null) throw new IllegalStateException("No X509 trust manager");
        for (int index = 0; index < args.length; index++) {
            X509Certificate[] chain;
            try (FileInputStream input = new FileInputStream(args[index])) {
                chain = CertificateFactory.getInstance("X.509").generateCertificates(input).toArray(new X509Certificate[0]);
            }
            boolean rejected = false;
            try {
                manager.checkServerTrusted(chain, "EC".equals(chain[0].getPublicKey().getAlgorithm()) ? "ECDHE_ECDSA" : "RSA");
            } catch (CertificateException error) {
                if (!error.toString().contains("unable to find valid certification path")) throw error;
                rejected = true;
            }
            if (rejected != (index > 0)) throw new IllegalStateException("Unexpected trust result: " + args[index]);
        }
        System.out.println("Private CA accepted; fourteen incident chains rejected by the active Java trust store.");
    }
}
